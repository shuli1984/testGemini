package models_test

import (
	"gemini-demo/internal/config"
	"gemini-demo/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAndSaveSiteConfig(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 1. Get initial site config from seed data
	siteConfig, err := models.GetSiteConfig(db, "en", "en")
	assert.NoError(t, err)
	assert.Equal(t, "My Awesome Website", siteConfig.Title)
	assert.Equal(t, "en", siteConfig.DefaultLanguage)

	// 2. Modify and save the site config
	siteConfig.Title = "My Updated Website"
	siteConfig.Navigation = []config.NavigationItem{
		{Label: "Home", Value: "/"},
		{Label: "About", Value: "/about"},
	}
	err = models.SaveSiteConfig(db, siteConfig, "en")
	assert.NoError(t, err)

	// 3. Get the config again and verify the changes
	updatedSiteConfig, err := models.GetSiteConfig(db, "en", "en")
	assert.NoError(t, err)
	assert.Equal(t, "My Updated Website", updatedSiteConfig.Title)
	assert.Len(t, updatedSiteConfig.Navigation, 2)
	assert.Equal(t, "Home", updatedSiteConfig.Navigation[0].Label)
}

func TestGetAndSaveSetting(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 1. Get a setting that doesn't exist
	value, err := models.GetSettingValue(db, "non-existent-key", "en")
	assert.NoError(t, err)
	assert.Empty(t, value)

	// 2. Save a new setting
	err = models.SaveSetting(db, "my-key", "my-value", "en")
	assert.NoError(t, err)

	// 3. Get the setting and verify the value
	value, err = models.GetSettingValue(db, "my-key", "en")
	assert.NoError(t, err)
	assert.Equal(t, "my-value", value)

	// 4. Update the setting
	err = models.SaveSetting(db, "my-key", "my-updated-value", "en")
	assert.NoError(t, err)

	// 5. Get the setting again and verify the updated value
	value, err = models.GetSettingValue(db, "my-key", "en")
	assert.NoError(t, err)
	assert.Equal(t, "my-updated-value", value)
}

func TestGetSiteConfig_ErrorPaths(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("with invalid navigation json", func(t *testing.T) {
		// Save a malformed JSON string
		models.SaveSetting(db, "navigation", "invalid-json", "en")

		// Get the site config
		siteConfig, err := models.GetSiteConfig(db, "en", "en")
		assert.NoError(t, err) // The function should not return an error, but log it
		assert.Nil(t, siteConfig.Navigation) // Navigation should be nil
	})

	t.Run("with invalid maintenance_mode bool", func(t *testing.T) {
		// Save a non-boolean string
		db.Model(&models.Setting{}).Where("key = ?", "maintenance_mode").Update("value", "not-a-bool")

		// Get the site config
		siteConfig, err := models.GetSiteConfig(db, "en", "en")
		assert.NoError(t, err) // The function should not return an error
		assert.False(t, siteConfig.MaintenanceMode) // MaintenanceMode should be false
	})
}