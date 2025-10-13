package models

import (
	"encoding/json"
	"gemini-demo/internal/config"
	"log"
	"strconv"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetSiteConfig loads all settings from the database and populates a SiteConfig struct
// for a specific language, with a fallback to the default language.
func GetSiteConfig(db *gorm.DB, lang string, defaultLang string) (*config.SiteConfig, error) {
	siteConfig := &config.SiteConfig{}

	// 1. Fetch non-translatable settings
	var settings []Setting
	if err := db.Find(&settings).Error; err != nil {
		return nil, err
	}
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}
	siteConfig.Logo = settingsMap["site_logo"]
	siteConfig.Favicon = settingsMap["site_favicon"]
	siteConfig.DefaultLanguage = settingsMap["default_language"]
	siteConfig.Timezone = settingsMap["timezone"]
	siteConfig.HomePage = settingsMap["home_page"]
	siteConfig.GoogleAnalyticsID = settingsMap["google_analytics_id"]
	maintenanceMode, err := strconv.ParseBool(settingsMap["maintenance_mode"])
	if err == nil {
		siteConfig.MaintenanceMode = maintenanceMode
	}

	// 2. Fetch translations for default language
	var defaultTranslations []SettingTranslation
	if err := db.Where("language_code = ?", defaultLang).Find(&defaultTranslations).Error; err != nil {
		return nil, err
	}
	defaultTranslationsMap := make(map[string]string)
	for _, t := range defaultTranslations {
		defaultTranslationsMap[t.Key] = t.Value
	}

	// 3. Fetch translations for the requested language
	var langTranslations []SettingTranslation
	if err := db.Where("language_code = ?", lang).Find(&langTranslations).Error; err != nil {
		return nil, err
	}
	langTranslationsMap := make(map[string]string)
	for _, t := range langTranslations {
		langTranslationsMap[t.Key] = t.Value
	}

	// 4. Combine translations, with requested language overriding default
	getValue := func(key string) string {
		if val, ok := langTranslationsMap[key]; ok && val != "" {
			return val
		}
		return defaultTranslationsMap[key]
	}

	siteConfig.Title = getValue("site_title")
	siteConfig.Tagline = getValue("site_tagline")
	siteConfig.MetaDescription = getValue("meta_description")
	siteConfig.MetaKeywords = getValue("meta_keywords")
	siteConfig.MaintenanceMessage = getValue("maintenance_message")

	// Unmarshal JSON for navigation
	navJSON := getValue("navigation")
	if navJSON != "" {
		var navigation []config.NavigationItem
		if err := json.Unmarshal([]byte(navJSON), &navigation); err != nil {
			log.Printf("Error unmarshalling navigation JSON for lang %s: %v. JSON: %s", lang, err, navJSON)
		} else {
			siteConfig.Navigation = navigation
		}
	}

	return siteConfig, nil
}

// SaveSiteConfig saves a SiteConfig struct to the database for a specific language.
func SaveSiteConfig(db *gorm.DB, siteConfig *config.SiteConfig, lang string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 1. Save non-translatable settings
		nonTranslatable := map[string]string{
			"site_logo":           siteConfig.Logo,
			"site_favicon":        siteConfig.Favicon,
			"default_language":    siteConfig.DefaultLanguage,
			"timezone":            siteConfig.Timezone,
			"home_page":           siteConfig.HomePage,
			"google_analytics_id": siteConfig.GoogleAnalyticsID,
			"maintenance_mode":    strconv.FormatBool(siteConfig.MaintenanceMode),
		}

		for key, value := range nonTranslatable {
			setting := Setting{Key: key, Value: value}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoUpdates: clause.AssignmentColumns([]string{"value"})}).Create(&setting).Error; err != nil {
				return err
			}
		}

		// 2. Save translatable settings for the given language
		navJSON, err := json.Marshal(siteConfig.Navigation)
		if err != nil {
			return err
		}

		translatable := map[string]string{
			"site_title":          siteConfig.Title,
			"site_tagline":        siteConfig.Tagline,
			"meta_description":    siteConfig.MetaDescription,
			"meta_keywords":       siteConfig.MetaKeywords,
			"maintenance_message": siteConfig.MaintenanceMessage,
			"navigation":          string(navJSON),
		}

		for key, value := range translatable {
			translation := SettingTranslation{Key: key, LanguageCode: lang, Value: value}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}, {Name: "language_code"}}, DoUpdates: clause.AssignmentColumns([]string{"value"})}).Create(&translation).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetSettingValue retrieves a single translated setting value from the database.
func GetSettingValue(db *gorm.DB, key, lang string) (string, error) {
	var translation SettingTranslation
	if err := db.Where("key = ? AND language_code = ?", key, lang).First(&translation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil // Return empty string and no error if not found
		}
		return "", err
	}
	return translation.Value, nil
}

// SaveSetting saves a single translated setting value.
func SaveSetting(db *gorm.DB, key, value, lang string) error {
	translation := SettingTranslation{Key: key, LanguageCode: lang, Value: value}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}, {Name: "language_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&translation).Error
}