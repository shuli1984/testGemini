package models

import (
	"gemini-demo/internal/config"
	"gorm.io/gorm"
)

// DataStore defines the interface for all database operations
// required by the application's handlers.
type DataStore interface {
	GetSiteData() (*Site, error)
	GetPageData(name string, lang string, defaultLang string) (*Page, error)
	GetAllPages(lang string, defaultLang string) ([]Page, error)
	CreatePage(page *Page) error
	UpdatePage(page *Page) error // Add this line
	UpdatePageTranslation(pageID uint, translation *PageTranslation) error
	DeletePage(name string) error
	GetDashboardData() (*DashboardData, error)
	GetCoreSolutions(lang string, defaultLang string) ([]Page, error)
	GetSiteConfig() (*config.SiteConfig, error)
	SaveSiteConfig(siteConfig *config.SiteConfig) error
}

// DBStore is a GORM implementation of the DataStore interface.
type DBStore struct {
	DB *gorm.DB
}

// NewDBStore creates a new DBStore.
func NewDBStore(db *gorm.DB) DataStore {
	return &DBStore{DB: db}
}

func (s *DBStore) GetSiteData() (*Site, error) {
	return GetSiteData(s.DB)
}

func (s *DBStore) GetPageData(name string, lang string, defaultLang string) (*Page, error) {
	return GetPageData(s.DB, name, lang, defaultLang)
}

func (s *DBStore) GetAllPages(lang string, defaultLang string) ([]Page, error) {
	return GetAllPages(s.DB, lang, defaultLang)
}

func (s *DBStore) CreatePage(page *Page) error {
	return CreatePage(s.DB, page)
}

func (s *DBStore) UpdatePage(page *Page) error {
	return UpdatePage(s.DB, page)
}

func (s *DBStore) UpdatePageTranslation(pageID uint, translation *PageTranslation) error {
	return UpdatePageTranslation(s.DB, pageID, translation)
}

func (s *DBStore) DeletePage(name string) error {
	return DeletePage(s.DB, name)
}

func (s *DBStore) GetDashboardData() (*DashboardData, error) {
	return GetDashboardData(s.DB)
}

func (s *DBStore) GetCoreSolutions(lang string, defaultLang string) ([]Page, error) {
	return GetCoreSolutions(s.DB, lang, defaultLang)
}

func (s *DBStore) GetSiteConfig() (*config.SiteConfig, error) {
	return GetSiteConfig(s.DB)
}

func (s *DBStore) SaveSiteConfig(siteConfig *config.SiteConfig) error {
	return SaveSiteConfig(s.DB, siteConfig)
}
