package handler_test

import (
	"fmt"
	"gemini-demo/internal/config"
	"gemini-demo/internal/models"
)

// MockErrorStore is a mock implementation of the DataStore interface that always returns an error.
type MockErrorStore struct{}

func (m *MockErrorStore) GetPageData(name, lang, defaultLang string) (*models.Page, error) {
	return nil, fmt.Errorf("database error")
}

func (m *MockErrorStore) GetAllPages(lang, defaultLang string) ([]models.Page, error) {
	return nil, fmt.Errorf("database error")
}

func (m *MockErrorStore) CreatePage(page *models.Page) error {
	return fmt.Errorf("database error")
}

func (m *MockErrorStore) UpdatePage(page *models.Page) error {
	return fmt.Errorf("database error")
}

func (m *MockErrorStore) UpdatePageTranslation(pageID uint, translation *models.PageTranslation) error {
	return fmt.Errorf("database error")
}

func (m *MockErrorStore) DeletePage(name string) error {
	return fmt.Errorf("database error")
}

func (m *MockErrorStore) GetCoreSolutions(lang, defaultLang string) ([]models.Page, error) {
	return nil, fmt.Errorf("database error")
}

func (m *MockErrorStore) GetSiteConfig(lang, defaultLang string) (*config.SiteConfig, error) {
	return nil, fmt.Errorf("database error")
}

func (m *MockErrorStore) SaveSiteConfig(siteConfig *config.SiteConfig, lang string) error {
	return fmt.Errorf("database error")
}

func (m *MockErrorStore) GetPageCount() (int64, error) {
	return 0, fmt.Errorf("database error")
}

func (m *MockErrorStore) GetSettingValue(key, lang string) (string, error) {
	return "", fmt.Errorf("database error")
}

func (m *MockErrorStore) GetRecentPages(limit int, lang, defaultLang string) ([]models.Page, error) {
	return nil, fmt.Errorf("database error")
}

func (m *MockErrorStore) CreateLoginLog(log *models.LoginLog) error {
	return fmt.Errorf("database error")
}

func (m *MockErrorStore) GetRecentLoginLogs(limit int) ([]models.LoginLog, error) {
	return nil, fmt.Errorf("database error")
}

// MockSuccessStore is a mock implementation of the DataStore interface that always succeeds.
type MockSuccessStore struct {
	Pages []models.Page
	SiteConfig *config.SiteConfig
}

func (m *MockSuccessStore) GetPageData(name, lang, defaultLang string) (*models.Page, error) {
	for _, p := range m.Pages {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("page not found")
}

func (m *MockSuccessStore) GetAllPages(lang, defaultLang string) ([]models.Page, error) {
	return m.Pages, nil
}

func (m *MockSuccessStore) CreatePage(page *models.Page) error {
	m.Pages = append(m.Pages, *page)
	return nil
}

func (m *MockSuccessStore) UpdatePage(page *models.Page) error {
	for i, p := range m.Pages {
		if p.ID == page.ID {
			m.Pages[i] = *page
			return nil
		}
	}
	return fmt.Errorf("page not found")
}

func (m *MockSuccessStore) UpdatePageTranslation(pageID uint, translation *models.PageTranslation) error {
	for i, p := range m.Pages {
		if p.ID == pageID {
			m.Pages[i].Content = *translation
			return nil
		}
	}
	return fmt.Errorf("page not found")
}

func (m *MockSuccessStore) DeletePage(name string) error {
	for i, p := range m.Pages {
		if p.Name == name {
			m.Pages = append(m.Pages[:i], m.Pages[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("page not found")
}

func (m *MockSuccessStore) GetCoreSolutions(lang, defaultLang string) ([]models.Page, error) {
	var coreSolutions []models.Page
	for _, p := range m.Pages {
		if p.IsCoreSolution {
			coreSolutions = append(coreSolutions, p)
		}
	}
	return coreSolutions, nil
}

func (m *MockSuccessStore) GetSiteConfig(lang, defaultLang string) (*config.SiteConfig, error) {
	return m.SiteConfig, nil
}

func (m *MockSuccessStore) SaveSiteConfig(siteConfig *config.SiteConfig, lang string) error {
	m.SiteConfig = siteConfig
	return nil
}

func (m *MockSuccessStore) GetPageCount() (int64, error) {
	return int64(len(m.Pages)), nil
}

func (m *MockSuccessStore) GetSettingValue(key, lang string) (string, error) {
	return "", nil
}

func (m *MockSuccessStore) GetRecentPages(limit int, lang, defaultLang string) ([]models.Page, error) {
	if len(m.Pages) > limit {
		return m.Pages[:limit], nil
	}
	return m.Pages, nil
}

func (m *MockSuccessStore) CreateLoginLog(log *models.LoginLog) error {
	return nil
}

func (m *MockSuccessStore) GetRecentLoginLogs(limit int) ([]models.LoginLog, error) {
	return []models.LoginLog{}, nil
}
