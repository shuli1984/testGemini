package models

import (
	"encoding/json"
	"gemini-demo/internal/config"
	"strconv"

	"gorm.io/gorm"
)

// GetSiteConfig loads all settings from the database and populates a SiteConfig struct.
func GetSiteConfig(db *gorm.DB) (*config.SiteConfig, error) {
	var settings []Setting
	if err := db.Find(&settings).Error; err != nil {
		return nil, err
	}

	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}

	siteConfig := &config.SiteConfig{}

	// Map simple key-value pairs
	siteConfig.Title = settingsMap["site_title"]
	siteConfig.Tagline = settingsMap["site_tagline"]
	siteConfig.Logo = settingsMap["site_logo"]
	siteConfig.Favicon = settingsMap["site_favicon"]
	siteConfig.DefaultLanguage = settingsMap["default_language"]
	siteConfig.Timezone = settingsMap["timezone"]
	siteConfig.HomePage = settingsMap["home_page"]
	siteConfig.MetaDescription = settingsMap["meta_description"]
	siteConfig.MetaKeywords = settingsMap["meta_keywords"]
	siteConfig.GoogleAnalyticsID = settingsMap["google_analytics_id"]
	siteConfig.MaintenanceMessage = settingsMap["maintenance_message"]

	// Map boolean value
	maintenanceMode, err := strconv.ParseBool(settingsMap["maintenance_mode"])
	if err == nil {
		siteConfig.MaintenanceMode = maintenanceMode
	}

	// Unmarshal JSON for navigation
	if navJSON, ok := settingsMap["navigation"]; ok && navJSON != "" {
		var navigation []config.NavigationItem
		if err := json.Unmarshal([]byte(navJSON), &navigation); err != nil {
			return nil, err
		}
		siteConfig.Navigation = navigation
	}

	return siteConfig, nil
}

// SaveSiteConfig saves a SiteConfig struct to the key-value settings table in the database.
func SaveSiteConfig(db *gorm.DB, siteConfig *config.SiteConfig) error {
	// Use a transaction to ensure all settings are saved or none are.
	return db.Transaction(func(tx *gorm.DB) error {
		// Simple key-value pairs
		settingsToSave := map[string]string{
			"site_title":          siteConfig.Title,
			"site_tagline":        siteConfig.Tagline,
			"site_logo":           siteConfig.Logo,
			"site_favicon":        siteConfig.Favicon,
			"default_language":    siteConfig.DefaultLanguage,
			"timezone":            siteConfig.Timezone,
			"home_page":           siteConfig.HomePage,
			"meta_description":    siteConfig.MetaDescription,
			"meta_keywords":       siteConfig.MetaKeywords,
			"google_analytics_id": siteConfig.GoogleAnalyticsID,
			"maintenance_mode":    strconv.FormatBool(siteConfig.MaintenanceMode),
			"maintenance_message": siteConfig.MaintenanceMessage,
		}

		for key, value := range settingsToSave {
			// Using .Save() on a struct with a primary key will perform an upsert (update or insert).
			if err := tx.Save(&Setting{Key: key, Value: value}).Error; err != nil {
				return err
			}
		}

		// Marshal and save navigation
		navJSON, err := json.Marshal(siteConfig.Navigation)
		if err != nil {
			return err
		}
		navSetting := Setting{Key: "navigation", Value: string(navJSON)}
		if err := tx.Save(&navSetting).Error; err != nil {
			return err
		}

		return nil
	})
}
