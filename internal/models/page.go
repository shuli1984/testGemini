package models

import (
	"encoding/json"
	"gemini-demo/internal/util"
	"html/template"
	"io/ioutil"
	"log"
	"path/filepath"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Page represents a logical page, which groups together translations.
type Page struct {
	gorm.Model
	Name           string `gorm:"uniqueIndex"` // Unique slug for the page group
	IsCoreSolution bool
	Icon           string
	// Content is a temporary field to hold the content of a specific language for display.
	// It is NOT stored in the 'pages' table.
	Content PageTranslation `gorm:"-"`
}

// PageTranslation holds the content for a page in a specific language.
type PageTranslation struct {
	gorm.Model
	PageID       uint   `gorm:"uniqueIndex:idx_page_lang"`
	LanguageCode string `gorm:"uniqueIndex:idx_page_lang"`
	Title        string
	Description  string
	Keywords     string // For SEO meta keywords
	Message      template.HTML
}

func AutoMigrateAndSeed(db *gorm.DB) error {
	log.Printf("Attempting AutoMigrate for Page and PageTranslation...")
	err := db.AutoMigrate(&Page{}, &PageTranslation{})
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

// GetPageData retrieves a page by its name and populates it with the content from the specified language.
// It falls back to the default language if the requested language is not available.
func GetPageData(db *gorm.DB, name string, lang string, defaultLang string) (*Page, error) {
	var page Page
	// First, find the main page entry
	if err := db.Where("name = ?", name).First(&page).Error; err != nil {
		return nil, err // This will be gorm.ErrRecordNotFound if no page with that name exists
	}

	// Now, try to find the translation for the requested language
	var translation PageTranslation
	err := db.Where("page_id = ? AND language_code = ?", page.ID, lang).First(&translation).Error

	// If translation is not found for the specific language, try the default language
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			err_default := db.Where("page_id = ? AND language_code = ?", page.ID, defaultLang).First(&translation).Error
			if err_default != nil {
				// If default also not found, return an error.
				return nil, err_default
			}
		} else {
			// A different database error occurred
			return nil, err
		}
	}

	// We found a translation, populate it into the non-persisted Content field
	page.Content = translation

	return &page, nil
}

// GetAllPages retrieves all pages, populating each with the content from the specified language.
func GetAllPages(db *gorm.DB, lang string, defaultLang string) ([]Page, error) {
	var pages []Page
	if err := db.Find(&pages).Error; err != nil {
		return nil, err
	}

	if len(pages) == 0 {
		return []Page{}, nil
	}

	// Extract page IDs for the query
	pageIDs := make([]uint, len(pages))
	for i, p := range pages {
		pageIDs[i] = p.ID
	}

	// Fetch all relevant translations in one go
	var translations []PageTranslation
	// We want translations that match the desired lang OR the default lang for the pages we are interested in
	db.Where("page_id IN ? AND language_code IN ?", pageIDs, []string{lang, defaultLang}).Find(&translations)

	// Create a map for easy lookup: pageID -> lang -> translation
	transMap := make(map[uint]map[string]PageTranslation)
	for _, t := range translations {
		if _, ok := transMap[t.PageID]; !ok {
			transMap[t.PageID] = make(map[string]PageTranslation)
		}
		transMap[t.PageID][t.LanguageCode] = t
	}

	// Populate the content for each page
	for i := range pages {
		if pageTranslations, ok := transMap[pages[i].ID]; ok {
			// Prioritize the requested language
			if translation, ok := pageTranslations[lang]; ok {
				pages[i].Content = translation
			} else if translation, ok := pageTranslations[defaultLang]; ok {
				// Fallback to the default language
				pages[i].Content = translation
			}
			// If no translation is found for either, the Content will remain the zero value
		}
	}

	return pages, nil
}

// GetCoreSolutions retrieves all pages marked as core solutions,
// populating each with the content from the specified language.
func GetCoreSolutions(db *gorm.DB, lang string, defaultLang string) ([]Page, error) {
	var pages []Page
	// Find pages where IsCoreSolution is true
	if err := db.Where("is_core_solution = ?", true).Find(&pages).Error; err != nil {
		return nil, err
	}

	if len(pages) == 0 {
		return []Page{}, nil
	}

	// The rest of the logic is identical to GetAllPages for fetching translations
	pageIDs := make([]uint, len(pages))
	for i, p := range pages {
		pageIDs[i] = p.ID
	}

	var translations []PageTranslation
	db.Where("page_id IN ? AND language_code IN ?", pageIDs, []string{lang, defaultLang}).Find(&translations)

	transMap := make(map[uint]map[string]PageTranslation)
	for _, t := range translations {
		if _, ok := transMap[t.PageID]; !ok {
			transMap[t.PageID] = make(map[string]PageTranslation)
		}
		transMap[t.PageID][t.LanguageCode] = t
	}

	for i := range pages {
		if pageTranslations, ok := transMap[pages[i].ID]; ok {
			// Prioritize the requested language
			if translation, ok := pageTranslations[lang]; ok {
				pages[i].Content = translation
			} else if translation, ok := pageTranslations[defaultLang]; ok {
				// Fallback to the default language
				pages[i].Content = translation
			}
		}
	}

	return pages, nil
}

// CreatePage creates a new page and its initial translation in a single transaction.
func CreatePage(db *gorm.DB, page *Page) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Create the main page entry
		if err := tx.Create(page).Error; err != nil {
			return err
		}

		// Now that the page has an ID, create the associated translation.
		// The translation is expected to be in the Content field.
		if page.Content.LanguageCode != "" {
			page.Content.PageID = page.ID
			if err := tx.Create(&page.Content).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// UpdatePageTranslation updates or creates a translation for a given page.
func UpdatePageTranslation(db *gorm.DB, pageID uint, translation *PageTranslation) error {
	translation.PageID = pageID
	// Use Clauses(clause.OnConflict) to perform an "upsert".
	// If a translation for the given PageID and LanguageCode exists, it will be updated.
	// Otherwise, a new one will be created.
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "page_id"}, {Name: "language_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "description", "keywords", "message"}),
	}).Create(translation).Error
}

// DeletePage deletes a page and all its associated translations.
func DeletePage(db *gorm.DB, name string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var page Page
		// Find the page by name
		if err := tx.Where("name = ?", name).First(&page).Error; err != nil {
			return err
		}

		// Delete all associated translations
		if err := tx.Where("page_id = ?", page.ID).Delete(&PageTranslation{}).Error; err != nil {
			return err
		}

		// Delete the page itself
		if err := tx.Delete(&page).Error; err != nil {
			return err
		}

		return nil
	})
}

// UpdatePage updates the non-translation fields of a Page.
func UpdatePage(db *gorm.DB, page *Page) error {
	return db.Model(page).Select("IsCoreSolution", "Icon").Updates(Page{IsCoreSolution: page.IsCoreSolution, Icon: page.Icon}).Error
}


