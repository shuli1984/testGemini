package models

import (
	"encoding/json"
	"fmt"
	"gemini-demo/internal/util"
	"io/ioutil"
	"log"
	"path/filepath"

	"gorm.io/gorm"
)

func AutoMigrateAndSeed(db *gorm.DB) error {
	log.Printf("Attempting AutoMigrate for Page, PageTranslation, Setting, SettingTranslation and LoginLog...")
	err := db.AutoMigrate(&Page{}, &PageTranslation{}, &Setting{}, &SettingTranslation{}, &LoginLog{})
	if err != nil {
		log.Printf("AutoMigrate failed: %v", err)
		return err
	}
	log.Printf("AutoMigrate successful.")

	// Read seed data from JSON file
	seedFilePath := filepath.Join(util.ProjectRoot(""), "data", "seed_data.json")
	seedFileContent, err := ioutil.ReadFile(seedFilePath)
	if err != nil {
		log.Printf("Failed to read seed file: %v", err)
		return err
	}

	var seedData SeedData
	err = json.Unmarshal(seedFileContent, &seedData)
	if err != nil {
		log.Printf("Failed to unmarshal seed data: %v", err)
		return err
	}

	// Seed pages if not exists
	var pageCount int64
	db.Model(&Page{}).Count(&pageCount)
	log.Printf("Current page count: %d", pageCount)
	if pageCount == 0 {
		if err := seedPages(db, &seedData); err != nil {
			return err
		}
	}

	// Seed site settings if not exists
	var settingCount int64
	db.Model(&Setting{}).Count(&settingCount)
	if settingCount == 0 {
		if err := seedSettings(db, &seedData); err != nil {
			return err
		}
	}

	return nil
}

func seedPages(db *gorm.DB, seedData *SeedData) error {
	log.Printf("Seeding pages...")
	for _, seedPage := range seedData.Pages {
		page := Page{
			Name:           seedPage.Name,
			IsCoreSolution: seedPage.IsCoreSolution,
			Icon:           seedPage.Icon,
		}
		db.Create(&page)

		for _, trans := range seedPage.Translations {
			translation := PageTranslation{
				PageID:       page.ID,
				LanguageCode: trans.LanguageCode,
				Title:        trans.Title,
				Description:  trans.Description,
				Keywords:     trans.Keywords,
				Message:      trans.Message,
			}
			db.Create(&translation)
		}
	}
	log.Printf("Seeding pages complete.")
	return nil
}

func seedSettings(db *gorm.DB, seedData *SeedData) error {
	log.Printf("Seeding settings from seed_data.json...")

	// Seed non-translatable settings
	if len(seedData.Settings) > 0 {
		if err := db.Create(&seedData.Settings).Error; err != nil {
			log.Printf("Failed to seed settings: %v", err)
			return err
		}
	}

	// Seed translatable settings
	for _, s := range seedData.SettingTranslations {
		var valueStr string
		if s.Key == "navigation" {
			jsonBytes, err := json.Marshal(s.Value)
			if err != nil {
				log.Printf("Failed to marshal navigation seed data for lang %s: %v", s.LanguageCode, err)
				continue // or return err
			}
			valueStr = string(jsonBytes)
		} else {
			valueStr = fmt.Sprintf("%v", s.Value)
		}

		translation := SettingTranslation{
			Key:          s.Key,
			LanguageCode: s.LanguageCode,
			Value:        valueStr,
		}
		if err := db.Create(&translation).Error; err != nil {
			log.Printf("Failed to seed setting translation for key %s, lang %s: %v", s.Key, s.LanguageCode, err)
		}
	}

	log.Printf("Seeding settings complete.")
	return nil
}
