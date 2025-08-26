package main

import (
	"fmt"
	"gemini-demo/internal/database"
	"gemini-demo/internal/models"
	"gemini-demo/internal/server"
	"log"
	"html/template"
	"path/filepath"
	"os"
	"strings"
	"gemini-demo/internal/util"

	"github.com/spf13/viper"
	"github.com/gorilla/csrf" // New import
)

// parseTemplates walks the templates directory and parses all .html files.
func parseTemplates() (*template.Template, error) {
	projectRoot := util.ProjectRoot("")
	var templateFiles []string
	err := filepath.Walk(filepath.Join(projectRoot, "templates"), func(path string, info os.FileInfo, err error) error {
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
		return nil, fmt.Errorf("no HTML templates found in %s", filepath.Join(projectRoot, "templates"))
	}

	tmpl, err := template.ParseFiles(templateFiles...)
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %w", err)
	}
	return tmpl, nil
}

func main() {
	// Load configuration
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	// Initialize database
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

	// Parse templates once at startup
	parsedTemplates, err := parseTemplates()
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	// CSRF Protection Setup
	csrfKey := viper.GetString("auth.csrf_key")
	if csrfKey == "" {
		log.Fatalf("CSRF key not found in config. Please set csrf.key")
	}
	// Ensure the key is 32 bytes long for HMAC-SHA256
	if len(csrfKey) < 32 {
		log.Fatalf("CSRF key must be at least 32 bytes long")
	}
	csrfMiddleware := csrf.Protect(
		[]byte(csrfKey),
		csrf.FieldName("csrf_token"), // Default is "csrf_token"
		csrf.HttpOnly(true),
		csrf.Secure(viper.GetBool("server.secure_cookies")), // Set to true in production with HTTPS
		csrf.SameSite(csrf.SameSiteStrictMode),
	)

	// Create and start server
	srv := server.New(db, parsedTemplates, csrfMiddleware) // Pass csrfMiddleware
	addr := viper.GetString("server.address")
	srv.Addr = addr

	fmt.Printf("Server is listening on %s\n", addr)
	log.Fatal(srv.ListenAndServe())
}
