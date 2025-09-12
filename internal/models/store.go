package models

import "gorm.io/gorm"

// DataStore defines the interface for all database operations
// required by the application's handlers.
type DataStore interface {
	GetSiteData() (*Site, error)
	GetPageData(name string) (*Page, error)
	GetAllPages() ([]Page, error)
	GetCoreSolutions() ([]Page, error)
	UpdatePage(page *Page) error
	CreatePage(page *Page) error
	DeletePage(name string) error
	GetDashboardData() (*DashboardData, error)
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

func (s *DBStore) GetPageData(name string) (*Page, error) {
	return GetPageData(s.DB, name)
}

func (s *DBStore) GetAllPages() ([]Page, error) {
	return GetAllPages(s.DB)
}

func (s *DBStore) GetCoreSolutions() ([]Page, error) {
	return GetCoreSolutions(s.DB)
}

func (s *DBStore) UpdatePage(page *Page) error {
	return page.UpdatePage(s.DB)
}

func (s *DBStore) CreatePage(page *Page) error {
	return page.CreatePage(s.DB)
}

func (s *DBStore) DeletePage(name string) error {
	return DeletePage(s.DB, name)
}

func (s *DBStore) GetDashboardData() (*DashboardData, error) {
	return GetDashboardData(s.DB)
}