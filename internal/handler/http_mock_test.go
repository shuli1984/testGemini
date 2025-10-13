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

func (m *MockErrorStore) GetPagesByIDs(ids []uint, lang string, defaultLang string) ([]models.Page, error) {
	return nil, fmt.Errorf("database error")
}

func (m *MockErrorStore) SaveSetting(key, value, lang string) error {
	return fmt.Errorf("database error")
}

// MockSuccessStore is a mock implementation of the DataStore interface that always succeeds.
type MockSuccessStore struct {
	Pages      []models.Page
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

func (m *MockSuccessStore) GetPagesByIDs(ids []uint, lang string, defaultLang string) ([]models.Page, error) {
	return []models.Page{}, nil
}

func (m *MockSuccessStore) SaveSetting(key, value, lang string) error {
	return nil
}

// CustomMockStore allows for custom mock behavior
type CustomMockStore struct {
	GetAllPagesFunc           func(lang, defaultLang string) ([]models.Page, error)
	GetPageDataFunc           func(name, lang, defaultLang string) (*models.Page, error)
	GetPagesByIDsFunc         func(ids []uint, lang string, defaultLang string) ([]models.Page, error)
	GetSiteConfigFunc         func(lang, defaultLang string) (*config.SiteConfig, error)
	UpdatePageFunc            func(page *models.Page) error
	UpdatePageTranslationFunc func(pageID uint, translation *models.PageTranslation) error
	CreatePageFunc            func(page *models.Page) error
	DeletePageFunc            func(name string) error
	GetSettingValueFunc       func(key, lang string) (string, error)
	CreateLoginLogFunc        func(log *models.LoginLog) error
	SaveSiteConfigFunc        func(siteConfig *config.SiteConfig, lang string) error
	SaveSettingFunc           func(key, value, lang string) error
	GetPageCountFunc          func() (int64, error)
	GetRecentPagesFunc        func(limit int, lang, defaultLang string) ([]models.Page, error)
	GetRecentLoginLogsFunc    func(limit int) ([]models.LoginLog, error)
	// Add other methods as needed
}

func (m *CustomMockStore) GetPageData(name, lang, defaultLang string) (*models.Page, error) {
	if m.GetPageDataFunc != nil {
		return m.GetPageDataFunc(name, lang, defaultLang)
	}
	return nil, fmt.Errorf("GetPageDataFunc not implemented")
}

func (m *CustomMockStore) GetAllPages(lang, defaultLang string) ([]models.Page, error) {
	if m.GetAllPagesFunc != nil {
		return m.GetAllPagesFunc(lang, defaultLang)
	}
	return nil, fmt.Errorf("GetAllPagesFunc not implemented")
}

func (m *CustomMockStore) GetPagesByIDs(ids []uint, lang string, defaultLang string) ([]models.Page, error) {
	if m.GetPagesByIDsFunc != nil {
		return m.GetPagesByIDsFunc(ids, lang, defaultLang)
	}
	return nil, fmt.Errorf("GetPagesByIDsFunc not implemented")
}

func (m *CustomMockStore) CreatePage(page *models.Page) error {
	if m.CreatePageFunc != nil {
		return m.CreatePageFunc(page)
	}
	return fmt.Errorf("CreatePageFunc not implemented")
}

func (m *CustomMockStore) UpdatePage(page *models.Page) error {
	if m.UpdatePageFunc != nil {
		return m.UpdatePageFunc(page)
	}
	return fmt.Errorf("UpdatePageFunc not implemented")
}

func (m *CustomMockStore) UpdatePageTranslation(pageID uint, translation *models.PageTranslation) error {
	if m.UpdatePageTranslationFunc != nil {
		return m.UpdatePageTranslationFunc(pageID, translation)
	}
	return fmt.Errorf("UpdatePageTranslationFunc not implemented")
}

func (m *CustomMockStore) DeletePage(name string) error {
	if m.DeletePageFunc != nil {
		return m.DeletePageFunc(name)
	}
	return fmt.Errorf("DeletePageFunc not implemented")
}

func (m *CustomMockStore) GetCoreSolutions(lang, defaultLang string) ([]models.Page, error) {
	return nil, nil
}

func (m *CustomMockStore) GetSiteConfig(lang, defaultLang string) (*config.SiteConfig, error) {
	if m.GetSiteConfigFunc != nil {
		return m.GetSiteConfigFunc(lang, defaultLang)
	}
	return nil, fmt.Errorf("GetSiteConfigFunc not implemented")
}

func (m *CustomMockStore) SaveSiteConfig(siteConfig *config.SiteConfig, lang string) error {
	if m.SaveSiteConfigFunc != nil {
		return m.SaveSiteConfigFunc(siteConfig, lang)
	}
	return fmt.Errorf("SaveSiteConfigFunc not implemented")
}

func (m *CustomMockStore) GetPageCount() (int64, error) {
	if m.GetPageCountFunc != nil {
		return m.GetPageCountFunc()
	}
	return 0, fmt.Errorf("GetPageCountFunc not implemented")
}

func (m *CustomMockStore) GetSettingValue(key, lang string) (string, error) {
	if m.GetSettingValueFunc != nil {
		return m.GetSettingValueFunc(key, lang)
	}
	return "", fmt.Errorf("GetSettingValueFunc not implemented")
}

func (m *CustomMockStore) SaveSetting(key, value, lang string) error {
	if m.SaveSettingFunc != nil {
		return m.SaveSettingFunc(key, value, lang)
	}
	return fmt.Errorf("SaveSettingFunc not implemented")
}

func (m *CustomMockStore) GetRecentPages(limit int, lang, defaultLang string) ([]models.Page, error) {
	if m.GetRecentPagesFunc != nil {
		return m.GetRecentPagesFunc(limit, lang, defaultLang)
	}
	return nil, fmt.Errorf("GetRecentPagesFunc not implemented")
}

func (m *CustomMockStore) CreateLoginLog(log *models.LoginLog) error {
	if m.CreateLoginLogFunc != nil {
		return m.CreateLoginLogFunc(log)
	}
	return fmt.Errorf("CreateLoginLogFunc not implemented")
}

func (m *CustomMockStore) GetRecentLoginLogs(limit int) ([]models.LoginLog, error) {
	if m.GetRecentLoginLogsFunc != nil {
		return m.GetRecentLoginLogsFunc(limit)
	}
	return nil, fmt.Errorf("GetRecentLoginLogsFunc not implemented")
}
