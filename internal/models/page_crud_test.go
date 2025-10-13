package models_test

import (
	"gemini-demo/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestPageCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	var newPageID uint

	// --- Create Test ---
	t.Run("Create Page", func(t *testing.T) {
		newPage := &models.Page{
			Name:           "test_page",
			IsCoreSolution: false,
			Icon:           "test-icon",
			Content: models.PageTranslation{
				LanguageCode: "en",
				Title:        "Test Title",
				Description:  "Test Description",
				Message:      "Test Message",
			},
		}
		err := models.CreatePage(db, newPage)
		if err != nil {
			t.Fatalf("failed to create page: %v", err)
		}
		if newPage.ID == 0 {
			t.Errorf("expected ID to be set, got 0")
		}
		newPageID = newPage.ID
	})

	// --- Read Test ---
	t.Run("Read Page", func(t *testing.T) {
		page, err := models.GetPageData(db, "test_page", "en", "en")
		if err != nil {
			t.Fatalf("failed to read page: %v", err)
		}
		if page.Name != "test_page" {
			t.Errorf("page name mismatch: got %s, want test_page", page.Name)
		}
		if page.Content.Title != "Test Title" {
			t.Errorf("page title mismatch: got %s, want Test Title", page.Content.Title)
		}
		if page.Content.Message != "Test Message" {
			t.Errorf("page message mismatch: got %s, want Test Message", page.Content.Message)
		}
	})

	// --- Update Test ---
	t.Run("Update Page Translation", func(t *testing.T) {
		updatedTranslation := &models.PageTranslation{
			LanguageCode: "en",
			Title:        "Updated Title",
			Description:  "Updated Description",
			Message:      "Updated Message",
		}

		err := models.UpdatePageTranslation(db, newPageID, updatedTranslation)
		if err != nil {
			t.Fatalf("failed to update page translation: %v", err)
		}

		updatedPage, err := models.GetPageData(db, "test_page", "en", "en")
		if err != nil {
			t.Fatalf("failed to read updated page: %v", err)
		}
		if updatedPage.Content.Title != "Updated Title" {
			t.Errorf("updated page title mismatch: got %q, want %q", updatedPage.Content.Title, "Updated Title")
		}
	})

	// --- Delete Test ---
	t.Run("Delete Page", func(t *testing.T) {
		err := models.DeletePage(db, "test_page")
		if err != nil {
			t.Fatalf("failed to delete page: %v", err)
		}

		_, err = models.GetPageData(db, "test_page", "en", "en")
		if err != gorm.ErrRecordNotFound {
			t.Errorf("expected record not found after delete, got %v", err)
		}
	})

	t.Run("Create Page without content", func(t *testing.T) {
		newPage := &models.Page{
			Name:           "test_page_no_content",
			IsCoreSolution: false,
			Icon:           "test-icon-no-content",
		}
		err := models.CreatePage(db, newPage)
		assert.NoError(t, err)

		_, err = models.GetPageData(db, "test_page_no_content", "en", "en")
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)

		// Verify the page was created, even without translation
		var p models.Page
		err = db.Where("name = ?", "test_page_no_content").First(&p).Error
		assert.NoError(t, err)
		assert.Equal(t, "test_page_no_content", p.Name)
	})
}
