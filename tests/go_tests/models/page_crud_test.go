package models_test

import (
	"fmt"
	"gemini-demo/internal/database"
	"gemini-demo/internal/models" // Added import for the models package
	"gemini-demo/tests/testutil"
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func TestPageCRUD(t *testing.T) {
	// Setup database for testing
	testutil.SetupViper()

	// Use a unique database file for each test run to avoid conflicts
	testDBName := fmt.Sprintf("./crud_test_%d.db", time.Now().UnixNano())
	viper.Set("database.dsn", testDBName)

	db, sqlDB, err := database.InitDB()
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}
	defer func() {
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()
	defer os.Remove(testDBName) // Clean up the unique database file

	// AutoMigrate for this test
	err = db.AutoMigrate(&models.Page{}) // Updated to models.Page{}
	if err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}

	// --- Create Test ---
	t.Run("Create Page", func(t *testing.T) {
		newPage := &models.Page{Name: "test_page", Title: "Test Title", Message: "Test Message"} // Updated to models.Page{}
		result := db.Create(newPage)
		if result.Error != nil {
			t.Fatalf("failed to create page: %v", result.Error)
		}
		if newPage.ID == 0 {
			t.Errorf("expected ID to be set, got 0")
		}
	})

	// --- Read Test ---
	t.Run("Read Page", func(t *testing.T) {
		page, err := models.GetPageData(db, "test_page") // Updated to models.GetPageData
		if err != nil {
			t.Fatalf("failed to read page: %v", err)
		}
		if page.Name != "test_page" || page.Title != "Test Title" || page.Message != "Test Message" {
			t.Errorf("read page data mismatch: got %+v", page)
		}
	})

	// --- Update Test ---
	t.Run("Update Page", func(t *testing.T) {
		page, err := models.GetPageData(db, "test_page") // Updated to models.GetPageData
		if err != nil {
			t.Fatalf("failed to get page for update: %v", err)
		}
		page.Title = "Updated Title"
		result := db.Save(page)
		if result.Error != nil {
			t.Fatalf("failed to update page: %v", result.Error)
		}

		updatedPage, err := models.GetPageData(db, "test_page") // Updated to models.GetPageData
		if err != nil {
			t.Fatalf("failed to read updated page: %v", err)
		}
		if updatedPage.Title != "Updated Title" {
			t.Errorf("updated page title mismatch: got %q, want %q", updatedPage.Title, "Updated Title")
		}
	})

	// --- Delete Test ---
	t.Run("Delete Page", func(t *testing.T) {
		page, err := models.GetPageData(db, "test_page") // Updated to models.GetPageData
		if err != nil {
			t.Fatalf("failed to get page for delete: %v", err)
		}
		result := db.Delete(page)
		if result.Error != nil {
			t.Fatalf("failed to delete page: %v", result.Error)
		}

		_, err = models.GetPageData(db, "test_page") // Updated to models.GetPageData
		if err != gorm.ErrRecordNotFound {
			t.Errorf("expected record not found after delete, got %v", err)
		}
	})
}

func TestAutoMigrateAndSeed_AutoMigrateError(t *testing.T) {
	// Setup a temporary read-only database to trigger an error in AutoMigrate
	testDBName := fmt.Sprintf("./crud_test_%d.db", time.Now().UnixNano())
	file, err := os.Create(testDBName)
	if err != nil {
		t.Fatalf("failed to create temp db file: %v", err)
	}
	file.Close()

	// Change file permissions to read-only
	if err := os.Chmod(testDBName, 0400); err != nil {
		t.Fatalf("failed to change file permissions: %v", err)
	}
	defer os.Remove(testDBName)

	viper.Set("database.dsn", testDBName)
	db, sqlDB, err := database.InitDB()
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}
	defer func() {
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	err = models.AutoMigrateAndSeed(db)
	if err == nil {
		t.Errorf("expected an error from AutoMigrateAndSeed with a read-only database, but got nil")
	}
}
