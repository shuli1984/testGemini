package models

import (
	"gemini-demo/internal/config"
	"gorm.io/gorm"
)

// DataStore defines the interface for all database operations
// required by the application's handlers.
type DataStore interface {
	GetPageData(name string, lang string, defaultLang string) (*Page, error)
	GetAllPages(lang string, defaultLang string) ([]Page, error)
	CreatePage(page *Page) error
	UpdatePage(page *Page) error // Add this line
	UpdatePageTranslation(pageID uint, translation *PageTranslation) error
	DeletePage(name string) error
	GetCoreSolutions(lang string, defaultLang string) ([]Page, error)
	GetSiteConfig(lang string, defaultLang string) (*config.SiteConfig, error)
	SaveSiteConfig(siteConfig *config.SiteConfig, lang string) error
	GetPageCount() (int64, error)
	GetSettingValue(key, lang string) (string, error)
	GetRecentPages(limit int, lang string, defaultLang string) ([]Page, error)
	CreateLoginLog(log *LoginLog) error
	GetRecentLoginLogs(limit int) ([]LoginLog, error)
}

// DBStore is a GORM implementation of the DataStore interface.
type DBStore struct {
	DB *gorm.DB
}

// NewDBStore creates a new DBStore.
func NewDBStore(db *gorm.DB) DataStore {
	return &DBStore{DB: db}
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

func (s *DBStore) GetPageCount() (int64, error) {
	var pageCount int64
	err := s.DB.Model(&Page{}).Count(&pageCount).Error
	return pageCount, err
}

func (s *DBStore) GetCoreSolutions(lang string, defaultLang string) ([]Page, error) {
	return GetCoreSolutions(s.DB, lang, defaultLang)
}

func (s *DBStore) GetSiteConfig(lang string, defaultLang string) (*config.SiteConfig, error) {
	return GetSiteConfig(s.DB, lang, defaultLang)
}

func (s *DBStore) SaveSiteConfig(siteConfig *config.SiteConfig, lang string) error {
	return SaveSiteConfig(s.DB, siteConfig, lang)
}

func (s *DBStore) GetSettingValue(key, lang string) (string, error) {
	return GetSettingValue(s.DB, key, lang)
}

func (s *DBStore) GetRecentPages(limit int, lang string, defaultLang string) ([]Page, error) {
	return GetRecentPages(s.DB, limit, lang, defaultLang)
}

func (s *DBStore) CreateLoginLog(log *LoginLog) error {
	return s.DB.Create(log).Error
}

func (s *DBStore) GetRecentLoginLogs(limit int) ([]LoginLog, error) {
	var logs []LoginLog
	err := s.DB.Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
}