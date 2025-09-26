package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"gemini-demo/internal/config"
	"gemini-demo/internal/database"
	"gemini-demo/internal/i18n"
	"gemini-demo/internal/logger"
	"gemini-demo/internal/models"
	"gemini-demo/internal/server"
	"gemini-demo/internal/translator"
	"gemini-demo/internal/util"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/csrf"
	"github.com/joho/godotenv"
)

var projectRootFlag string
var debugFlag bool
var loadConfig = config.LoadConfig

func debugLog(format string, v ...interface{}) {
	if debugFlag {
		log.Printf(format, v...)
	}
}

// generateRandomKey creates a random key of the specified length (in bytes)
// and returns it as a hex-encoded string.
func generateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("application returned an error: %v", err)
	}
}

func run(args []string) error {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file, using system environment variables: %v", err)
	}

	fs := flag.NewFlagSet("gemini-demo", flag.ContinueOnError)
	fs.StringVar(&projectRootFlag, "project-root", "", "Absolute path to the project root directory")
	fs.BoolVar(&debugFlag, "debug", false, "Enable debug logging")
	if err := fs.Parse(args); err != nil {
		return err
	}
	log.Printf("Debug mode enabled: %t", debugFlag) // Add this line

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("Error reading config file, %s", err)
	}

	// In debug mode, if keys are not set, generate temporary ones.
	if debugFlag {
		// Username and Password are now handled directly by os.Getenv in auth.go
		// and do not need to be set via cfg.Auth here.

		// Ensure SessionKey is set
		if cfg.Auth.SessionKey == "" {
			return fmt.Errorf("Session key not found in config or environment. Please set auth.session_key or SESSION_KEY environment variable.")
		}
		// Ensure CSRFKey is set
		if cfg.Auth.CSRFKey == "" {
			return fmt.Errorf("CSRF key not found in config or environment. Please set auth.csrf_key or CSRF_KEY environment variable.")
		}

		debugLog("Using session key: %s", cfg.Auth.SessionKey)
	}

	i18nBasePath := filepath.Join(util.ProjectRoot(""), "data", "i18n")
	i18nTranslator := i18n.NewTranslator(i18nBasePath, cfg.I18n.DefaultLanguage)
	if err := i18nTranslator.LoadTranslations(); err != nil {
		return fmt.Errorf("Failed to load translations: %v", err)
	}

	apiTranslator, err := translator.New(context.Background(), cfg.Translator.Type, cfg.Translator.APIKey)
	if err != nil {
		return fmt.Errorf("Failed to create translator: %v", err)
	}

	db, sqlDB, err := database.InitDB(cfg.Database.Type, cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	defer func() {
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	err = models.AutoMigrateAndSeed(db)
	if err != nil {
		return fmt.Errorf("failed to auto migrate and seed models: %v", err)
	}

	// Load dynamic site settings from the database
	siteConfigFromDB, err := models.GetSiteConfig(db, cfg.I18n.DefaultLanguage, cfg.I18n.DefaultLanguage)
	if err != nil {
		return fmt.Errorf("failed to load site settings from database: %v", err)
	}
	// Merge DB settings into the main config struct
	cfg.Site = *siteConfigFromDB

	parsedTemplates, err := util.ParseTemplates(i18nTranslator, projectRootFlag)
	if err != nil {
		return fmt.Errorf("failed to parse templates: %v", err)
	}

	if cfg.Auth.CSRFKey == "" {
		return fmt.Errorf("CSRF key not found in config. Please set auth.csrf_key")
	}
	if len(cfg.Auth.CSRFKey) < 32 {
		return fmt.Errorf("CSRF key must be at least 32 bytes long")
	}
	debugLog("CSRF Key used: %s", cfg.Auth.CSRFKey)

	if len(cfg.Auth.TrustedOrigins) == 0 {
		log.Printf("Warning: No CSRF trusted origins configured. This may lead to 'origin invalid' errors.")
	}
	debugLog("Configured Trusted Origins: %v", cfg.Auth.TrustedOrigins)

	logRequestMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debugLog("Request before CSRF: URL: %s, Host: %s, Origin: %s, Referer: %s", r.URL.String(), r.Host, r.Header.Get("Origin"), r.Header.Get("Referer"))
			next.ServeHTTP(w, r)
		})
	}

	csrfMiddleware := csrf.Protect(
		[]byte(cfg.Auth.CSRFKey),
		csrf.HttpOnly(true),
		csrf.Secure(true),
		csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.Path("/"),
		csrf.TrustedOrigins(cfg.Auth.TrustedOrigins),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debugLog("CSRF Error: Handler triggered for request to %s", r.URL.Path) // Made conditional
			// Log the CSRF failure reason
			if err := csrf.FailureReason(r); err != nil { // Pass the request directly
				debugLog("CSRF Error: Failure Reason: %v", err) // Made conditional
			}
			debugLog("CSRF Error: Expected Token (from csrf.Token(r)): %s", csrf.Token(r)) // Made conditional
			debugLog("CSRF Error: Origin: %s, Referer: %s", r.Header.Get("Origin"), r.Header.Get("Referer")) // Made conditional
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"message": "Forbidden - CSRF token invalid."})
		})),
	)

	// Add this line to debug the session key being used
	debugLog("AuthService Session Key being used: %s", cfg.Auth.SessionKey)

	startTime := time.Now()
	inMemoryLogger := logger.NewInMemoryLogCollector(10) // Capacity of 10 errors

	srv := server.New(cfg, db, parsedTemplates, func(h http.Handler) http.Handler {
		// Conditionally apply PlaintextHTTPRequest for HTTP connections
		wrappedHandler := logRequestMiddleware(csrfMiddleware(h))
		if !strings.HasPrefix(cfg.Server.Address, "https://") { // Assuming non-HTTPS is HTTP
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wrappedHandler.ServeHTTP(w, csrf.PlaintextHTTPRequest(r))
			})
		}
		return wrappedHandler
	}, i18nTranslator, apiTranslator, debugLog, debugFlag, inMemoryLogger, startTime) // Pass the translator and debug flag
	srv.Addr = cfg.Server.Address

	fmt.Printf("Server is listening on %s\n", srv.Addr)
	return srv.ListenAndServe()
}
