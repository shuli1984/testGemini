package main

import (
	"flag"
	"fmt"
	"gemini-demo/internal/database"
	"gemini-demo/internal/models"
	"gemini-demo/internal/server"
	"log"
	"html/template"
	"path/filepath"
	"os"
	"strings"
	"net/http"
	"gemini-demo/internal/util"

	v "github.com/spf13/viper"
	"github.com/gorilla/csrf"
)

var projectRootFlag string

func parseTemplates() (*template.Template, error) {
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

	tmpl, err := template.ParseFiles(templateFiles...)
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %w", err)
	}
	return tmpl, nil
}

func main() {
	flag.StringVar(&projectRootFlag, "project-root", "", "Absolute path to the project root directory")
	flag.Parse()

	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.SetConfigType("yml")
	if err := v.ReadInConfig(); err != nil {
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

	parsedTemplates, err := parseTemplates()
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	csrfKey := v.GetString("auth.csrf_key")
	if csrfKey == "" {
		log.Fatalf("CSRF key not found in config. Please set csrf.key")
	}
	if len(csrfKey) < 32 {
		log.Fatalf("CSRF key must be at least 32 bytes long")
	}
	log.Printf("CSRF Key used: %s", csrfKey)

	logRequestMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Request before CSRF: URL: %s, Host: %s, Origin: %s, Referer: %s", r.URL.String(), r.Host, r.Header.Get("Origin"), r.Header.Get("Referer"))
			next.ServeHTTP(w, r)
		})
	}

	csrfMiddleware := csrf.Protect(
		[]byte(csrfKey),
		csrf.HttpOnly(true),
		csrf.Secure(false), // Set to true in production with HTTPS
		csrf.SameSite(csrf.SameSiteLaxMode), // Use Lax for same-site applications
		csrf.Path("/"), // Set the cookie path to the root
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("CSRF Error: Handler triggered for request to %s", r.URL.Path)
			log.Printf("CSRF Error: Failed Token: %s", r.Header.Get("X-CSRF-Token"))
			log.Printf("CSRF Error: Origin: %s, Referer: %s", r.Header.Get("Origin"), r.Header.Get("Referer"))
			http.Error(w, "Forbidden - CSRF token invalid.", http.StatusForbidden)
		})),
	)

	srv := server.New(db, parsedTemplates, func(h http.Handler) http.Handler {
		return logRequestMiddleware(csrfMiddleware(h))
	})
	addr := v.GetString("server.address")
	srv.Addr = addr

	fmt.Printf("Server is listening on %s\n", addr)
	log.Fatal(srv.ListenAndServe())
}
