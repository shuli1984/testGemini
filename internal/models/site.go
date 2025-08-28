package models

import (
	"encoding/json"
	"gorm.io/gorm"
	"html/template"
	"os"
	"path/filepath"
	"gemini-demo/internal/util"
)

// SiteSetting represents a key-value pair for site-wide settings in the database.
type SiteSetting struct {
    gorm.Model
    Key   string `gorm:"uniqueIndex"`
    Value string
}

// MenuItemDB represents a menu item stored in the database.
type MenuItemDB struct {
    gorm.Model
    URL   string
    Text  string
    Order int `gorm:"default:0"`
}

// MenuItem represents a single item in the navigation menu (for application logic).
type MenuItem struct {
    URL  string
    Text string
}

// CarouselItem represents a single item in the hero carousel.
type CarouselItem struct {
    Title         template.HTML
    Description   template.HTML
    ButtonText    string
    ButtonLink    string
    BackgroundImage string
    Active        bool // Add this line
}

// Site represents the overall site configuration and data.
type Site struct {
    Title       string
    Description string
    CanonicalURL string
    ImageURL    string
    HeroTitle   template.HTML // Keep for backward compatibility or if still used elsewhere
    HeroDescription template.HTML // Keep for backward compatibility or if still used elsewhere
    Keywords    string // New SEO field
    Robots      string // New SEO field

    MenuItems []MenuItem // Navigation menu items
    CarouselItems []CarouselItem // New field for hero carousel items
}

// DashboardData combines Site data and PageCount for the dashboard.
type DashboardData struct {
	*Site // Embed Site struct
	PageCount int64
}

// GetDashboardData fetches all necessary data for the admin dashboard.
func GetDashboardData(db *gorm.DB) (*DashboardData, error) {
	siteData, err := GetSiteData(db)
	if err != nil {
		return nil, err
	}

	var pageCount int64
	db.Model(&Page{}).Count(&pageCount)

	return &DashboardData{
		Site:      siteData,
		PageCount: pageCount,
	}, nil
}

func loadSeedData() (*SeedData, error) {
	projectRoot := util.ProjectRoot("")
	filePath := filepath.Join(projectRoot, "data", "seed_data.json")
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var seedData SeedData
	err = json.Unmarshal(file, &seedData)
	if err != nil {
		return nil, err	}

	return &seedData, nil
}

// GetSiteData fetches site-wide data, including menu items, from the database.
func GetSiteData(db *gorm.DB) (*Site, error) {
    // Auto-migrate the tables (for development/initial setup)
    err := db.AutoMigrate(&SiteSetting{}, &MenuItemDB{})
    if err != nil {
        return nil, err
    }

    site := &Site{}

    // Fetch site settings
    var settings []SiteSetting
    if err := db.Find(&settings).Error; err != nil {
        return nil, err
    }

    // Populate Site struct from settings
    for _, setting := range settings {
        switch setting.Key {
        case "Title":
            site.Title = setting.Value
        case "Description":
            site.Description = setting.Value
        case "CanonicalURL":
            site.CanonicalURL = setting.Value
        case "ImageURL":
            site.ImageURL = setting.Value
        case "HeroTitle":
            site.HeroTitle = template.HTML(setting.Value)
        case "HeroDescription":
            site.HeroDescription = template.HTML(setting.Value)
        case "Keywords": // New SEO field
            site.Keywords = setting.Value
        case "Robots":   // New SEO field
            site.Robots = setting.Value
        }
    }

    // Fetch menu items
    var menuItemsDB []MenuItemDB
    if err := db.Order("`order` asc").Find(&menuItemsDB).Error; err != nil {
        return nil, err
    }

    // Convert MenuItemDB to MenuItem
    for _, itemDB := range menuItemsDB {
        site.MenuItems = append(site.MenuItems, MenuItem{
            URL:  itemDB.URL,
            Text: itemDB.Text,
        })
    }

    // If no settings found, populate with defaults from seed_data.json and save (for initial run)
    if len(settings) == 0 {
        seedData, err := loadSeedData()
        if err != nil {
            return nil, err
        }

        // Save default settings
        for _, setting := range seedData.SiteSettings {
            db.Create(&setting)
        }

        // Save default menu items
        for _, item := range seedData.MenuItems {
            db.Create(&item)
        }

        // Re-fetch to ensure the Site struct is fully populated from DB after defaults are saved
        return GetSiteData(db)
    }

    return site, nil
}