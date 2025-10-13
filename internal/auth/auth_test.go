package auth_test

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gemini-demo/internal/auth"
	"gemini-demo/internal/config"
	"gemini-demo/internal/handler"
	"gemini-demo/internal/i18n"
	"gemini-demo/internal/models"
	"gemini-demo/internal/translator"
	"gemini-demo/internal/util"

	"github.com/gorilla/sessions"
)

// MockTranslator is a mock implementation of the translator.Translator interface for testing.
type MockTranslator struct{}

func (m *MockTranslator) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	// For testing, just return the original text with language codes
	return fmt.Sprintf("%s (translated from %s to %s)", text, sourceLang, targetLang), nil
}

func (m *MockTranslator) Close() error {
	return nil
}

var _ translator.Translator = (*MockTranslator)(nil)

// MockStore is a mock implementation of the models.DataStore interface for testing.
type MockStore struct{}

func (m *MockStore) GetPageData(name string, lang string, defaultLang string) (*models.Page, error) {
	return nil, nil
}
func (m *MockStore) GetAllPages(lang string, defaultLang string) ([]models.Page, error) {
	return nil, nil
}
func (m *MockStore) CreatePage(page *models.Page) error {
	return nil
}
func (m *MockStore) UpdatePage(page *models.Page) error {
	return nil
}
func (m *MockStore) UpdatePageTranslation(pageID uint, translation *models.PageTranslation) error {
	return nil
}
func (m *MockStore) DeletePage(name string) error {
	return nil
}
func (m *MockStore) GetCoreSolutions(lang string, defaultLang string) ([]models.Page, error) {
	return nil, nil
}
func (m *MockStore) GetSiteConfig(lang string, defaultLang string) (*config.SiteConfig, error) {
	return &config.SiteConfig{}, nil
}
func (m *MockStore) SaveSiteConfig(siteConfig *config.SiteConfig, lang string) error {
	return nil
}
func (m *MockStore) GetPageCount() (int64, error) {
	return 0, nil
}
func (m *MockStore) GetSettingValue(key, lang string) (string, error) {
	return "", nil
}
func (m *MockStore) GetRecentPages(limit int, lang string, defaultLang string) ([]models.Page, error) {
	return nil, nil
}
func (m *MockStore) CreateLoginLog(log *models.LoginLog) error {
	return nil
}
func (m *MockStore) GetRecentLoginLogs(limit int) ([]models.LoginLog, error) {
	return nil, nil
}

func (m *MockStore) GetPagesByIDs(ids []uint, lang string, defaultLang string) ([]models.Page, error) {
	return nil, nil
}

func (m *MockStore) SaveSetting(key, value, lang string) error {
	return nil
}

var _ models.DataStore = (*MockStore)(nil)

func TestNewAuthService(t *testing.T) {
	t.Run("successful creation with valid key", func(t *testing.T) {
		authService := auth.NewAuthService("test-secret-key-for-sessions-32")
		if authService == nil {
			t.Error("auth.NewAuthService returned nil")
		}
	})

	t.Run("logs fatal if session key is missing", func(t *testing.T) {
		oldFatalLogger := auth.FatalLogger
		var fatalLogged bool
		auth.FatalLogger = func(v ...interface{}) {
			fatalLogged = true
		}
		defer func() { auth.FatalLogger = oldFatalLogger }() // Restore original auth.FatalLogger

		auth.NewAuthService("")

		if !fatalLogged {
			t.Error("expected auth.NewAuthService to log fatal, but it did not")
		}
	})
}

func TestAuthenticate(t *testing.T) {
	// Setup: Ensure environment variables are clean before and after tests
	originalAdminUsername := os.Getenv("ADMIN_USERNAME")
	originalAdminPassword := os.Getenv("ADMIN_PASSWORD")
	defer func() {
		os.Setenv("ADMIN_USERNAME", originalAdminUsername)
		os.Setenv("ADMIN_PASSWORD", originalAdminPassword)
	}()

	authService := auth.NewAuthService("test-secret-key-for-sessions-32")

	t.Run("successful authentication with env vars", func(t *testing.T) {
		os.Setenv("ADMIN_USERNAME", "testuser")
		os.Setenv("ADMIN_PASSWORD", "testpass")
		if !authService.Authenticate("testuser", "testpass") {
			t.Error("Authenticate failed for valid credentials with env vars")
		}
	})

	t.Run("successful authentication with default values", func(t *testing.T) {
		// Temporarily set environment variables for this test case
		os.Setenv("ADMIN_USERNAME", "admin")
		os.Setenv("ADMIN_PASSWORD", "password")
		defer os.Unsetenv("ADMIN_USERNAME") // Clean up after the test
		defer os.Unsetenv("ADMIN_PASSWORD") // Clean up after the test

		if !authService.Authenticate("admin", "password") {
			t.Error("Authenticate failed for valid credentials with default values")
		}
	})

	t.Run("successful authentication with fallback values", func(t *testing.T) {
		os.Unsetenv("ADMIN_USERNAME")
		os.Unsetenv("ADMIN_PASSWORD")

		if !authService.Authenticate("admin", "password") {
			t.Error("Authenticate failed for valid credentials with fallback values")
		}
	})

	t.Run("failed authentication - incorrect username", func(t *testing.T) {
		os.Setenv("ADMIN_USERNAME", "testuser")
		os.Setenv("ADMIN_PASSWORD", "testpass")
		if authService.Authenticate("wronguser", "testpass") {
			t.Error("Authenticate succeeded for incorrect username")
		}
	})

	t.Run("failed authentication - incorrect password", func(t *testing.T) {
		os.Setenv("ADMIN_USERNAME", "testuser")
		os.Setenv("ADMIN_PASSWORD", "testpass")
		if authService.Authenticate("testuser", "wrongpass") {
			t.Error("Authenticate succeeded for incorrect password")
		}
	})
}

// MockSessionStore is a mock implementation of the sessions.Store interface.
type MockSessionStore struct {
	GetFunc  func(r *http.Request, name string) (*sessions.Session, error)
	SaveFunc func(r *http.Request, w http.ResponseWriter, s *sessions.Session) error
	NewFunc  func(r *http.Request, name string) (*sessions.Session, error)
}

func (m *MockSessionStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return m.GetFunc(r, name)
}

func (m *MockSessionStore) New(r *http.Request, name string) (*sessions.Session, error) {
	if m.NewFunc != nil {
		return m.NewFunc(r, name)
	}
	return sessions.NewSession(m, name), nil
}

func (m *MockSessionStore) Save(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
	return m.SaveFunc(r, w, s)
}

func TestLoginLogoutIsLoggedIn(t *testing.T) {
	authService := auth.NewAuthService("test-secret-key-for-sessions-32")

	// Test Login
	t.Run("successful login", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		if err := authService.Login(rr, req); err != nil {
			t.Fatalf("Login failed: %v", err)
		}
		// Check if cookie is set
		cookies := rr.Result().Cookies()
		if len(cookies) == 0 {
			t.Error("No cookies set after login")
		}
		// Check IsLoggedIn
		loggedInReq := httptest.NewRequest("GET", "/", nil)
		for _, cookie := range cookies {
			loggedInReq.AddCookie(cookie)
		}
		if !authService.IsLoggedIn(loggedInReq) {
			t.Error("IsLoggedIn returned false after successful login")
		}
	})

	t.Run("login fails on session get error", func(t *testing.T) {
		mockStore := &MockSessionStore{
			GetFunc: func(r *http.Request, name string) (*sessions.Session, error) {
				return nil, fmt.Errorf("session get error")
			},
		}

		service := auth.NewAuthServiceWithStore(mockStore)
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		err := service.Login(rr, req)
		if err == nil {
			t.Error("Expected an error when session get fails, but got nil")
		}
	})

	t.Run("login fails on session save error", func(t *testing.T) {
		mockStore := &MockSessionStore{}
		session := sessions.NewSession(mockStore, "test-session")
		mockStore.GetFunc = func(r *http.Request, name string) (*sessions.Session, error) {
			return session, nil
		}
		mockStore.SaveFunc = func(r *http.Request, w http.ResponseWriter, s *sessions.Session) error {
			return fmt.Errorf("session save error")
		}
		service := auth.NewAuthServiceWithStore(mockStore)
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		err := service.Login(rr, req)
		if err == nil {
			t.Error("Expected an error when session save fails, but got nil")
		}
	})

	// Test Logout
	t.Run("successful logout", func(t *testing.T) {
		// First, log in
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		_ = authService.Login(rr, req)

		// Get the session cookie from the login response
		var sessionCookie *http.Cookie
		for _, cookie := range rr.Result().Cookies() {
			if cookie.Name == "gemini-session" {
				sessionCookie = cookie
				break
			}
		}
		if sessionCookie == nil {
			t.Fatal("session cookie not found after simulated login for logout test")
		}

		// Now, logout
		logoutRr := httptest.NewRecorder()
		logoutReq, _ := http.NewRequest("GET", "/", nil)
		logoutReq.AddCookie(sessionCookie)
		if err := authService.Logout(logoutRr, logoutReq); err != nil {
			t.Fatalf("Logout failed: %v", err)
		}

		// Check if cookie is expired
		cookies := logoutRr.Result().Cookies()
		foundExpiredCookie := false
		for _, cookie := range cookies {
			if cookie.Name == "gemini-session" && cookie.MaxAge < 0 {
				foundExpiredCookie = true
				break
			}
		}
		if !foundExpiredCookie {
			t.Error("Session cookie not expired after logout")
		}

		// Check IsLoggedIn after logout
		loggedOutReq := httptest.NewRequest("GET", "/", nil)
		// Do not add the expired cookie, or add a new one if needed to simulate a fresh state
		if authService.IsLoggedIn(loggedOutReq) {
			t.Error("IsLoggedIn returned true after successful logout")
		}
	})

	t.Run("logout fails on session get error", func(t *testing.T) {
		mockStore := &MockSessionStore{
			GetFunc: func(r *http.Request, name string) (*sessions.Session, error) {
				return nil, fmt.Errorf("session get error")
			},
		}
		service := auth.NewAuthServiceWithStore(mockStore)
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		err := service.Logout(rr, req)
		if err == nil {
			t.Error("Expected an error when session get fails, but got nil")
		}
	})

	// Test IsLoggedIn with no session cookie
	t.Run("IsLoggedIn returns false with no session cookie", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		if authService.IsLoggedIn(req) {
			t.Error("IsLoggedIn returned true with no session cookie")
		}
	})

	t.Run("IsLoggedIn returns false on session get error", func(t *testing.T) {
		mockStore := &MockSessionStore{
			GetFunc: func(r *http.Request, name string) (*sessions.Session, error) {
				return nil, fmt.Errorf("session get error")
			},
		}
		service := auth.NewAuthServiceWithStore(mockStore)
		req, _ := http.NewRequest("GET", "/", nil)
		if service.IsLoggedIn(req) {
			t.Error("Expected IsLoggedIn to return false on session get error")
		}
	})

	// Test IsLoggedIn with invalid session cookie (e.g., wrong value)
	t.Run("IsLoggedIn returns false with invalid session cookie", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "gemini-session", Value: "invalid-token"})
		if authService.IsLoggedIn(req) {
			t.Error("IsLoggedIn returned true with invalid session cookie")
		}
	})
}

func TestMiddleware(t *testing.T) {
	// Dummy handler to be wrapped by middleware
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Protected content")
	})

	authService := auth.NewAuthService("test-secret-key-for-sessions-32")

	// Test Middleware - Not logged in (should redirect)
	t.Run("middleware redirects if not logged in", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)
		authService.Middleware(dummyHandler).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusFound {
			t.Errorf("Middleware did not redirect: got %v want %v", status, http.StatusFound)
		}
		if location := rr.Header().Get("Location"); location != "/admin/login" {
			t.Errorf("Middleware redirected to wrong location: got %v want %v", location, "/admin/login")
		}
	})

	// Test Middleware - Logged in (should pass through)
	t.Run("middleware passes through if logged in", func(t *testing.T) {
		// Simulate login to get a valid session cookie
		loginRr := httptest.NewRecorder()
		loginReq, _ := http.NewRequest("POST", "/admin/login", strings.NewReader(`{"username":"admin","password":"password"}`))
		// Temporarily set env vars for login handler to work
		os.Setenv("ADMIN_USERNAME", "admin")
		os.Setenv("ADMIN_PASSWORD", "password")
		defer os.Unsetenv("ADMIN_USERNAME")
		defer os.Unsetenv("ADMIN_PASSWORD")

		// Create a dummy template.Template
		tmpl := template.New("test")

		// Construct absolute path to i18n directory
		i18nPath := filepath.Join(util.ProjectRoot(""), "data", "i18n")

		// Create a simple Translator
		translator := i18n.NewTranslator(i18nPath, "en")
		// Load translations (handle error if necessary)
		if err := translator.LoadTranslations(); err != nil {
			t.Fatalf("Failed to load translations: %v", err)
		}

		// Create a dummy DebugLog function
		debugLog := func(format string, v ...interface{}) {
			// Do nothing or print to console for debugging
			// fmt.Printf(format+"\n", v...)
		}

		h := &handler.Handler{
			AuthService:    authService,
			Templates:      map[string]*template.Template{"default": tmpl},
			I18n:           translator,
			API_Translator: &MockTranslator{}, // Initialize API_Translator with mock
			DebugLog:       debugLog,
			Store:          &MockStore{},
		}
		h.LoginHandler(loginRr, loginReq)

		var sessionCookie *http.Cookie
		for _, cookie := range loginRr.Result().Cookies() {
			if cookie.Name == "gemini-session" {
				sessionCookie = cookie
				break
			}
		}
		if sessionCookie == nil {
			t.Fatal("session cookie not found after simulated login for middleware test")
		}

		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.AddCookie(sessionCookie)

		authService.Middleware(dummyHandler).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Middleware did not pass through: got %v want %v", status, http.StatusOK)
		}
		if rr.Body.String() != "Protected content" {
			t.Errorf("Middleware returned wrong body: got %q want %q", rr.Body.String(), "Protected content")
		}
	})
}
