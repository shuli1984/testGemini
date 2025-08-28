package models_test

import (
	"gemini-demo/internal/database"
	"gemini-demo/internal/models" // Added import for the models package
	"os"
	"testing"

	"gorm.io/gorm"
)

func TestGetPageData(t *testing.T) {
	// Set up the database for testing
	db, sqlDB, err := database.InitDB("sqlite", "./gemini.db")
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}
	defer func() {
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()
	models.AutoMigrateAndSeed(db) // Updated to models.AutoMigrateAndSeed
	defer os.Remove("./gemini.db")

	t.Run("gets page data for existing page", func(t *testing.T) {
		page, err := models.GetPageData(db, "home") // Updated to models.GetPageData
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if page.Title != "Innovatech - AI & IT 解决方案" {
			t.Errorf("unexpected title: got %q, want %q", page.Title, "Innovatech - AI & IT 解决方案")
		}

		if page.Message != "Hello from the database!" {
			t.Errorf("unexpected message: got %q, want %q", page.Message, "Hello from the database!")
		}
	})

	t.Run("returns error for non-existing page", func(t *testing.T) {
		_, err := models.GetPageData(db, "non-existing-page") // Updated to models.GetPageData
		if err == nil {
			t.Fatal("expected an error, but got nil")
		}
		// Check for GORM's specific "record not found" error
		if err != gorm.ErrRecordNotFound {
			t.Errorf("expected GORM's ErrRecordNotFound, got %v", err)
		}
	})
}