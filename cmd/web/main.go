package main

import (
	"flag"
	"fmt"
	"gemini-demo/internal/database"
	"gemini-demo/internal/models"
	"gemini-demo/internal/server"
	"gemini-demo/internal/i18n" // New import for i18n
	"log"
	"html/template"
	"path/filepath"
	"os"
	"strings"
	"net/http"
	"gemini-demo/internal/util"

	"github.com/spf13/viper"
	"github.com/gorilla/csrf" // Uncomment this import
)

var projectRootFlag string
var debugFlag bool // Declare a global variable for debug status

// debugLog prints messages only if debugFlag is true
func debugLog(format string, v ...interface{}) {
	if debugFlag {
		log.Printf(format, v...)
	}
}

func parseTemplates(translator *i18n.Translator) (*template.Template, error) {
	var actualProjectRoot string
	if projectRootFlag != "" {
		actualProjectRoot = projectRootFlag
	} else {
		actualProjectRoot = util.ProjectRoot("")
	}

	var templateFiles []string
	err := filepath.Walk(filepath.Join(actualProjectRoot, "templates"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			templateFiles = append(templateFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking templates directory: %w", err)
	}

	if len(templateFiles) == 0 {
		return nil, fmt.Errorf("no HTML templates found in %s", filepath.Join(actualProjectRoot, "templates"))
	}

	// Create a FuncMap for templates
	funcMap := template.FuncMap{
		"T": func(lang, key string) string {
			// This is a placeholder T function for the template parser.
			// The actual translation will be provided by the T function in the template data.
			return translator.GetTranslation(lang, key) // Use the provided language for parsing
		},
		"hasPrefix": strings.HasPrefix,
	}

	tmpl := template.New("main").Funcs(funcMap)
	tmpl, err = tmpl.ParseFiles(templateFiles...)
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %w", err)
	}
	return tmpl, nil
}

func main() {
	flag.StringVar(&projectRootFlag, "project-root", "", "Absolute path to the project root directory")
	flag.BoolVar(&debugFlag, "debug", false, "Enable debug logging") // Add debug flag
	flag.Parse()

	// Initialize server.DebugLog
	server.DebugLog = debugLog // Use capitalized DebugLog

	// Initialize i18n translator
	i18nBasePath := filepath.Join(util.ProjectRoot(""), "data", "i18n")
	translator := i18n.NewTranslator(i18nBasePath, "en") // "en" as default language
	if err := translator.LoadTranslations(); err != nil {
		log.Fatalf("Failed to load translations: %v", err)
	}

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	db, sqlDB, err := database.InitDB()
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

	parsedTemplates, err := parseTemplates(translator)
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	csrfKey := viper.GetString("auth.csrf_key")
	if csrfKey == "" {
		log.Fatalf("CSRF key not found in config. Please set csrf.key")
	}
	if len(csrfKey) < 32 {
		log.Fatalf("CSRF key must be at least 32 bytes long")
	}
	debugLog("CSRF Key used: %s", csrfKey) // Made conditional

	// Read trusted origins from config
	trustedOrigins := viper.GetStringSlice("auth.trusted_origins")
	debugLog("Configured Trusted Origins: %v", trustedOrigins) // Add this line
	if len(trustedOrigins) == 0 {
		log.Printf("Warning: No CSRF trusted origins configured. This may lead to 'origin invalid' errors.")
	}

	logRequestMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debugLog("Request before CSRF: URL: %s, Host: %s, Origin: %s, Referer: %s", r.URL.String(), r.Host, r.Header.Get("Origin"), r.Header.Get("Referer")) // Made conditional
			next.ServeHTTP(w, r)
		})
	}

	csrfMiddleware := csrf.Protect(
		[]byte(csrfKey),
		csrf.HttpOnly(true),
		csrf.Secure(true), // Set to true for production (HTTPS)
		csrf.SameSite(csrf.SameSiteLaxMode), // Re-enable SameSite
		csrf.Path("/"), // Set the cookie path to the root
		csrf.TrustedOrigins(trustedOrigins), // Use trusted origins from config
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			debugLog("CSRF Error: Handler triggered for request to %s", r.URL.Path) // Made conditional
			// Log the CSRF failure reason
			if err := csrf.FailureReason(r); err != nil { // Pass the request directly
				debugLog("CSRF Error: Failure Reason: %v", err) // Made conditional
			}
			// Log the expected CSRF token (optional for production, but useful for debugging if issues arise)
			debugLog("CSRF Error: Expected Token (from csrf.Token(r)): %s", csrf.Token(r)) // Made conditional
			debugLog("CSRF Error: Origin: %s, Referer: %s", r.Header.Get("Origin"), r.Header.Get("Referer")) // Made conditional
			http.Error(w, "Forbidden - CSRF token invalid.", http.StatusForbidden)
		})),
	)

	addr := viper.GetString("server.address")
	srv := server.New(db, parsedTemplates, func(h http.Handler) http.Handler {
		// Conditionally apply PlaintextHTTPRequest for HTTP connections
		wrappedHandler := logRequestMiddleware(csrfMiddleware(h))
		if !strings.HasPrefix(addr, "https://") { // Assuming non-HTTPS is HTTP
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wrappedHandler.ServeHTTP(w, csrf.PlaintextHTTPRequest(r))
			})
		}
		return wrappedHandler
	}, translator) // Pass the translator
	srv.Addr = addr

	fmt.Printf("Server is listening on %s\n", addr)
	log.Fatal(srv.ListenAndServe())
}
