package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"gemini-demo/internal/config"
	"gemini-demo/internal/database"
	"gemini-demo/internal/i18n"
	"gemini-demo/internal/models"
	"gemini-demo/internal/server"
	"gemini-demo/internal/util"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/joho/godotenv"
)

var projectRootFlag string
var debugFlag bool

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
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file, using system environment variables: %v", err)
	}

	flag.StringVar(&projectRootFlag, "project-root", "", "Absolute path to the project root directory")
	flag.BoolVar(&debugFlag, "debug", false, "Enable debug logging")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	// In debug mode, if keys are not set, generate temporary ones.
	if debugFlag {
		if cfg.Auth.Username == "" {
			cfg.Auth.Username = "admin"
			debugLog("Temporary username set to 'admin'.")
		}
		if cfg.Auth.Password == "" {
			password, err := generateRandomKey(16)
			if err != nil {
				log.Fatalf("Failed to generate temporary password: %v", err)
			}
			cfg.Auth.Password = password
			debugLog("Temporary password set to '%s'.", password)
		}

		if cfg.Auth.SessionKey == "" {
			key, err := generateRandomKey(32)
			if err != nil {
				log.Fatalf("Failed to generate temporary session key: %v", err)
			}
			cfg.Auth.SessionKey = key
			debugLog("Generated temporary session key.")
		}
		if cfg.Auth.CSRFKey == "" {
			key, err := generateRandomKey(32)
			if err != nil {
				log.Fatalf("Failed to generate temporary CSRF key: %v", err)
			}
			cfg.Auth.CSRFKey = key
			debugLog("Generated temporary CSRF key.")
		}
		debugLog("Using session key: %s", cfg.Auth.SessionKey)
	}

	i18nBasePath := filepath.Join(util.ProjectRoot(""), "data", "i18n")
	translator := i18n.NewTranslator(i18nBasePath, "en")
	if err := translator.LoadTranslations(); err != nil {
		log.Fatalf("Failed to load translations: %v", err)
	}

	db, sqlDB, err := database.InitDB(cfg.Database.Type, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer func() {
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	err = models.AutoMigrateAndSeed(db)
	if err != nil {
		log.Fatalf("failed to auto migrate and seed models: %v", err)
	}

	parsedTemplates, err := util.ParseTemplates(translator, projectRootFlag)
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	if cfg.Auth.CSRFKey == "" {
		log.Fatalf("CSRF key not found in config. Please set auth.csrf_key")
	}
	if len(cfg.Auth.CSRFKey) < 32 {
		log.Fatalf("CSRF key must be at least 32 bytes long")
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
			// Log the expected CSRF token (optional for production, but useful for debugging if issues arise)
			debugLog("CSRF Error: Expected Token (from csrf.Token(r)): %s", csrf.Token(r)) // Made conditional
			debugLog("CSRF Error: Origin: %s, Referer: %s", r.Header.Get("Origin"), r.Header.Get("Referer")) // Made conditional
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"message": "Forbidden - CSRF token invalid."})
		})),
	)

	srv := server.New(cfg, db, parsedTemplates, func(h http.Handler) http.Handler {
		// Conditionally apply PlaintextHTTPRequest for HTTP connections
		wrappedHandler := logRequestMiddleware(csrfMiddleware(h))
		if !strings.HasPrefix(cfg.Server.Address, "https://") { // Assuming non-HTTPS is HTTP
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wrappedHandler.ServeHTTP(w, csrf.PlaintextHTTPRequest(r))
			})
		}
		return wrappedHandler
	}, translator, debugLog, debugFlag) // Pass the translator and debug flag
	srv.Addr = cfg.Server.Address

	fmt.Printf("Server is listening on %s\n", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
