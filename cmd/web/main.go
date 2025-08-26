package main

import (
	"fmt"
	"gemini-demo/internal/database"
	"gemini-demo/internal/models"
	"gemini-demo/internal/server"
	"log"
	"html/template" // New import
	"path/filepath" // New import
	"os"            // New import
	"strings"       // New import
	"gemini-demo/internal/util" // New import for ProjectRoot

	"github.com/spf13/viper"
)

// parseTemplates walks the templates directory and parses all .html files.
func parseTemplates() (*template.Template, error) {
	projectRoot := util.ProjectRoot("") // Use util.ProjectRoot
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

	// Create and start server
	srv := server.New(db, parsedTemplates) // Pass parsedTemplates
	addr := viper.GetString("server.address")
	srv.Addr = addr

	fmt.Printf("Server is listening on %s\n", addr)
	log.Fatal(srv.ListenAndServe())
}
