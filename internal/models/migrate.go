package models

import (
	"encoding/json"
	"gemini-demo/internal/util"
	"io/ioutil"
	"log"
	"path/filepath"

	"gorm.io/gorm"
)

func AutoMigrateAndSeed(db *gorm.DB) error {
	log.Printf("Attempting AutoMigrate for Page, PageTranslation, and Setting...")
	// Add Setting model to the migration
	err := db.AutoMigrate(&Page{}, &PageTranslation{}, &Setting{})
	if err != nil {
		log.Printf("AutoMigrate failed: %v", err)
		return err
	}
	log.Printf("AutoMigrate successful.")

	// Seed data if not exists
	var count int64
	db.Model(&Page{}).Count(&count)
	log.Printf("Current page count: %d", count)
	if count == 0 {
		log.Printf("Seeding data...")
		// Read seed data from JSON file
		seedFilePath := filepath.Join(util.ProjectRoot(""), "data", "seed_data.json")
		log.Printf("Seed file path: %s", seedFilePath)
		seedFileContent, err := ioutil.ReadFile(seedFilePath)
		if err != nil {
			log.Printf("Failed to read seed file: %v", err)
			return err
		}

		var seedData SeedData // Use the SeedData struct
		err = json.Unmarshal(seedFileContent, &seedData)
		if err != nil {
			log.Printf("Failed to unmarshal seed data: %v", err)
			return err
		}

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
		log.Printf("Seeding complete. New page count: %d", len(seedData.Pages))
	}
	return nil
}
