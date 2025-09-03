package models

import (
	"encoding/json" // Added
	"io/ioutil"     // Added
	"log"
	"path/filepath" // Added
	"gemini-demo/internal/util" // Added

	"gorm.io/gorm"
)

type Page struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex"`
	Title       string
	Description string
	Message     string
}

func AutoMigrateAndSeed(db *gorm.DB) error {
	log.Printf("Attempting AutoMigrate for Page...")
	err := db.AutoMigrate(&Page{})
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

		for _, page := range seedData.Pages { // Iterate over seedData.Pages
			db.Create(&page)
		}
		log.Printf("Seeding complete. New page count: %d", len(seedData.Pages)) // Use seedData.Pages
	}
	return nil
}

func GetPageData(db *gorm.DB, name string) (*Page, error) {
	var page Page
	result := db.Where("name = ?", name).First(&page)
	if result.Error != nil {
		log.Println(result.Error) // Consider returning error directly without logging here
		return nil, result.Error
	}
	return &page, nil
}

func (p *Page) UpdatePage(db *gorm.DB) error {
	result := db.Save(p)
	if result.Error != nil {
		log.Printf("Failed to update page %s: %v", p.Name, result.Error)
		return result.Error
	}
	return nil
}

func (p *Page) CreatePage(db *gorm.DB) error {
	result := db.Create(p)
	if result.Error != nil {
		log.Printf("Failed to create page %s: %v", p.Name, result.Error)
		return result.Error
	}
	return nil
}

func DeletePage(db *gorm.DB, name string) error {
	result := db.Unscoped().Where("name = ?", name).Delete(&Page{})
	if result.Error != nil {
		log.Printf("Failed to delete page %s: %v", name, result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func GetAllPages(db *gorm.DB) ([]Page, error) {
	var pages []Page
	result := db.Find(&pages)
	if result.Error != nil {
		log.Printf("Failed to get all pages: %v", result.Error)
		return nil, result.Error
	}
	return pages, nil
}

