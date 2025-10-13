package models_test

import (
	"gemini-demo/internal/models"
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestGetPageData(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("gets page data for existing page", func(t *testing.T) {
		page, err := models.GetPageData(db, "home", "en", "en")
		assert.NoError(t, err)
		assert.NotNil(t, page)
		assert.Equal(t, "Innovatech - AI & IT Solutions", page.Content.Title)
		assert.Equal(t, template.HTML("Hello from the database!"), page.Content.Message)
	})

	t.Run("returns error for non-existing page", func(t *testing.T) {
		_, err := models.GetPageData(db, "non-existing-page", "en", "en")
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("falls back to default language", func(t *testing.T) {
		// Assuming 'home' page has 'en' translation but not 'de'
		page, err := models.GetPageData(db, "home", "de", "en")
		assert.NoError(t, err)
		assert.NotNil(t, page)
		assert.Equal(t, "en", page.Content.LanguageCode)
		assert.Equal(t, "Innovatech - AI & IT Solutions", page.Content.Title)
	})

	t.Run("returns error if no translation found", func(t *testing.T) {
		// Create a page with no translations to test this
		newPage := &models.Page{Name: "no-translation-page"}
		db.Create(newPage)

		_, err := models.GetPageData(db, "no-translation-page", "de", "fr")
		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestGetAllPages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("gets all pages", func(t *testing.T) {
		pages, err := models.GetAllPages(db, "en", "en")
		assert.NoError(t, err)
		assert.Len(t, pages, 7)

		var homePage models.Page
		for _, p := range pages {
			if p.Name == "home" {
				homePage = p
				break
			}
		}
		assert.NotZero(t, homePage.ID)
		assert.Equal(t, "Innovatech - AI & IT Solutions", homePage.Content.Title)
	})

}

func TestGetCoreSolutions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("gets all core solutions", func(t *testing.T) {
		pages, err := models.GetCoreSolutions(db, "en", "en")
		assert.NoError(t, err)
		assert.Len(t, pages, 3)

		var aiMlPage models.Page
		for _, p := range pages {
			if p.Name == "ai_ml" {
				aiMlPage = p
				break
			}
		}
		assert.NotZero(t, aiMlPage.ID)
		assert.Equal(t, "AI & Machine Learning", aiMlPage.Content.Title)
	})

}

func TestUpdatePage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	page, err := models.GetPageData(db, "home", "en", "en")
	assert.NoError(t, err)
	assert.NotNil(t, page)

	page.IsCoreSolution = true
	page.Icon = "fa-star"
	page.FeaturedImage = "/new/image.jpg"

	err = models.UpdatePage(db, page)
	assert.NoError(t, err)

	updatedPage, err := models.GetPageData(db, "home", "en", "en")
	assert.NoError(t, err)
	assert.NotNil(t, updatedPage)
	assert.True(t, updatedPage.IsCoreSolution)
	assert.Equal(t, "fa-star", updatedPage.Icon)
	assert.Equal(t, "/new/image.jpg", updatedPage.FeaturedImage)
}

func TestGetRecentPages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("gets recent pages", func(t *testing.T) {
		page, err := models.GetPageData(db, "about", "en", "en")
		assert.NoError(t, err)
		page.FeaturedImage = "/latest.jpg"
		err = models.UpdatePage(db, page)
		assert.NoError(t, err)

		recentPages, err := models.GetRecentPages(db, 1, "en", "en")
		assert.NoError(t, err)
		assert.Len(t, recentPages, 1)
		assert.Equal(t, "about", recentPages[0].Name)
		assert.Equal(t, "/latest.jpg", recentPages[0].FeaturedImage)
	})

}

func TestGetPagesByIDs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("gets pages by existing IDs", func(t *testing.T) {
		page1, err := models.GetPageData(db, "home", "en", "en")
		assert.NoError(t, err)
		page2, err := models.GetPageData(db, "contact", "en", "en")
		assert.NoError(t, err)

		ids := []uint{page1.ID, page2.ID}
		pages, err := models.GetPagesByIDs(db, ids, "en", "en")
		assert.NoError(t, err)
		assert.Len(t, pages, 2)

		foundPage1 := false
		foundPage2 := false
		for _, p := range pages {
			if p.ID == page1.ID {
				foundPage1 = true
				assert.Equal(t, "home", p.Name)
			}
			if p.ID == page2.ID {
				foundPage2 = true
				assert.Equal(t, "contact", p.Name)
			}
		}
		assert.True(t, foundPage1)
		assert.True(t, foundPage2)
	})

	t.Run("returns empty slice for non-existent IDs", func(t *testing.T) {
		pages, err := models.GetPagesByIDs(db, []uint{999, 998}, "en", "en")
		assert.NoError(t, err)
		assert.Len(t, pages, 0)
	})

	t.Run("returns empty slice for empty ID slice", func(t *testing.T) {
		pages, err := models.GetPagesByIDs(db, []uint{}, "en", "en")
		assert.NoError(t, err)
		assert.Len(t, pages, 0)
	})
}
