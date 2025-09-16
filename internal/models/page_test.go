package models_test

import (
	"gemini-demo/internal/models"
	"testing"

	"gorm.io/gorm"
)

func TestGetPageData(t *testing.T) {
	// Set up the database for testing
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("gets page data for existing page", func(t *testing.T) {
		page, err := models.GetPageData(db, "home", "en", "en") // Updated to models.GetPageData
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if page.Content.Title != "Innovatech - AI & IT Solutions" {
			t.Errorf("unexpected title: got %q, want %q", page.Content.Title, "Innovatech - AI & IT Solutions")
		}

		if page.Content.Message != "Hello from the database!" {
			t.Errorf("unexpected message: got %q, want %q", page.Content.Message, "Hello from the database!")
		}
	})

	t.Run("returns error for non-existing page", func(t *testing.T) {
		_, err := models.GetPageData(db, "non-existing-page", "en", "en") // Updated to models.GetPageData
		if err == nil {
			t.Fatal("expected an error, but got nil")
		}
		// Check for GORM's specific "record not found" error
		if err != gorm.ErrRecordNotFound {
			t.Errorf("expected GORM's ErrRecordNotFound, got %v", err)
		}
	})
}