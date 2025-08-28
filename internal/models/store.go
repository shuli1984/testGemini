package models

import "gorm.io/gorm"

// DataStore defines the interface for all database operations
// required by the application's handlers.
type DataStore interface {
	GetSiteData() (*Site, error)
	GetPageData(name string) (*Page, error)
	GetAllPages() ([]Page, error)
	UpdatePage(page *Page) error
	GetDashboardData() (*DashboardData, error)
}

// DBStore is a GORM implementation of the DataStore interface.
type DBStore struct {
	DB *gorm.DB
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

func (s *DBStore) UpdatePage(page *Page) error {
	return page.UpdatePage(s.DB)
}

func (s *DBStore) GetDashboardData() (*DashboardData, error) {
	return GetDashboardData(s.DB)
}
