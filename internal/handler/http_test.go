package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gemini-demo/internal/auth"
	"gemini-demo/internal/config"
	"gemini-demo/internal/handler"
	"gemini-demo/internal/i18n"
	"gemini-demo/internal/logger"
	"gemini-demo/internal/models"
	"gemini-demo/internal/testutil"
	"gemini-demo/internal/util"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockAPITranslator is a mock implementation of the translator.Translator interface for testing.
type MockAPITranslator struct {
	TranslateTextFunc func(ctx context.Context, text, sourceLang, targetLang string) (string, error)
}

func (m *MockAPITranslator) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	if m.TranslateTextFunc != nil {
		return m.TranslateTextFunc(ctx, text, sourceLang, targetLang)
	}
	return fmt.Sprintf("translated: %s", text), nil
}

func (m *MockAPITranslator) Close() error {
	return nil
}

// setupTest creates a new in-memory DB, auth service, and template set for testing.
func setupTest(t *testing.T) *handler.Handler {
	t.Helper()
	tmpDir, cleanup, err := testutil.SetupTestEnv(t)
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	originalWD, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.Chdir(originalWD)
		cleanup()
	})

	// Now that the environment is set up, util.ProjectRoot() will work correctly.
	projectRoot := util.ProjectRoot("")

	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	assert.NoError(t, err)

	sqlDB, err := db.DB()
	assert.NoError(t, err)
	t.Cleanup(func() {
		sqlDB.Close()
	})

	// Auto-migrate and seed models
	err = models.AutoMigrateAndSeed(db)
	assert.NoError(t, err)

	authService := auth.NewAuthService("super-secret-key-for-testing")

	// Initialize i18n translator for testing
	i18nBasePath := filepath.Join(projectRoot, "data", "i18n")
	translator := i18n.NewTranslator(i18nBasePath, "en")
	err = translator.LoadTranslations()
	assert.NoError(t, err)

	// Parse templates from the temporary directory
	templatesMap, err := util.ParseTemplates(translator, projectRoot)
	assert.NoError(t, err)

	// Initialize handler
	h := &handler.Handler{
		Cfg: &config.Config{
			Static: config.StaticConfig{
				URLPrefix: "/static/",
				Dir:       "static",
			},
			Site: config.SiteConfig{
				DefaultLanguage: "en",
			},
		},
		Store:          models.NewDBStore(db),
		AuthService:    authService,
		Templates:      templatesMap,
		I18n:           translator,
		DebugLog:       func(format string, v ...interface{}) { t.Logf(format, v...) },
		API_Translator: &MockAPITranslator{},
		ErrorLogger:    logger.NewInMemoryLogCollector(10),
	}

	return h
}

// mockCSRF is a mock CSRF middleware that does nothing.
func mockCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add a dummy csrf token to the context, so the handler doesn't panic
		ctx := context.WithValue(r.Context(), csrf.TemplateTag, "dummy_token")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TestIndexHandler(t *testing.T) {
	h := setupTest(t)
	assert := assert.New(t)

	// Insert test data

	req, err := http.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "en") // Added this line
	assert.NoError(err)

	rr := httptest.NewRecorder()
	h.IndexHandler(rr, req)

	assert.Equal(http.StatusOK, rr.Code)
	assert.Contains(rr.Body.String(), "My Awesome Website")
}

func TestPageHandler_Success(t *testing.T) {
	h := setupTest(t)

	// Insert test data
	pageName := "test-page"

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", fmt.Sprintf("/page/%s", pageName), nil)
	assert.NoError(t, err)

	// Set URL variables for mux
	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	h.PageHandler(rr, req)

	// Assertions
	assert.Contains(t, rr.Body.String(), "Test Page")
	assert.Contains(t, rr.Body.String(), "Test Page Message")
}

func TestPageHandler_NotFound(t *testing.T) {
	h := setupTest(t)

	// Create a request for a non-existent page
	pageName := "non-existent-page"

	req, err := http.NewRequest("GET", fmt.Sprintf("/page/%s", pageName), nil)
	assert.NoError(t, err)

	// Set URL variables for mux
	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	h.PageHandler(rr, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "page_not_found"))
}

func TestPageHandler_InternalError(t *testing.T) {
	h := setupTest(t)
	h.Store = &MockErrorStore{}

	pageName := "any-page"
	req, err := http.NewRequest("GET", fmt.Sprintf("/page/%s", pageName), nil)
	assert.NoError(t, err)

	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	h.PageHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "internal_server_error"))
}

func TestUpdatePageHandler_Success(t *testing.T) {
	h := setupTest(t)

	// Mock data
	pageName := "home"
	updatedPayload := handler.PageUpdatePayload{
		Name:         pageName,
		Title:        "Updated Home Title",
		Description:  "Updated Home Description",
		Message:      "Updated Home Message",
		LanguageCode: "en",
	}

	// Create request body
	body, err := json.Marshal(updatedPayload)
	assert.NoError(t, err)

	// Create a request
	req, err := http.NewRequest("PUT", fmt.Sprintf("/pages/%s", pageName), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Set URL variables for mux
	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	h.UpdatePageHandler(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	var respBody map[string]string
	err = json.NewDecoder(rr.Body).Decode(&respBody)
	assert.NoError(t, err)
	assert.Equal(t, h.I18n.GetTranslation("en", "save_successful"), respBody["message"])
}

func TestUpdatePageHandler_NotFound(t *testing.T) {
	h := setupTest(t)

	pageName := "nonexistent"
	updatedPayload := handler.PageUpdatePayload{Name: pageName, Title: "New Title", LanguageCode: "en"}

	// Create request body
	body, err := json.Marshal(updatedPayload)
	assert.NoError(t, err)

	// Create a request
	req, err := http.NewRequest("PUT", fmt.Sprintf("/pages/%s", pageName), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Set URL variables for mux
	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	h.UpdatePageHandler(rr, req)

	// Assertions
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "page_not_found"))
}

func TestUpdatePageHandler_InvalidBody(t *testing.T) {
	h := setupTest(t)

	pageName := "home"
	// Create a request with invalid JSON body
	req, err := http.NewRequest("PUT", fmt.Sprintf("/pages/%s", pageName), strings.NewReader("invalid json"))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	h.UpdatePageHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "invalid_request_body"))
}

func TestUpdatePageHandler_NameMismatch(t *testing.T) {
	h := setupTest(t)

	pageName := "home"
	updatedPayload := handler.PageUpdatePayload{Name: "mismatch-name", Title: "New Title", LanguageCode: "en"}
	body, err := json.Marshal(updatedPayload)
	assert.NoError(t, err)

	req, err := http.NewRequest("PUT", fmt.Sprintf("/pages/%s", pageName), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	h.UpdatePageHandler(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "page_name_mismatch"))
}

func TestUpdatePageHandler_InternalError(t *testing.T) {
	h := setupTest(t)
	h.Store = &MockErrorStore{}

	pageName := "home"
	updatedPayload := handler.PageUpdatePayload{Name: pageName, Title: "New Title", LanguageCode: "en"}
	body, err := json.Marshal(updatedPayload)
	assert.NoError(t, err)

	req, err := http.NewRequest("PUT", fmt.Sprintf("/pages/%s", pageName), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	h.UpdatePageHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestLoginHandler_GET_NotLoggedIn(t *testing.T) {
	h := setupTest(t)

	r := mux.NewRouter()
	r.HandleFunc("/admin/login", h.LoginHandler)

	ts := httptest.NewServer(mockCSRF(r))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/admin/login")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	bodyString := string(bodyBytes)

	assert.Contains(t, bodyString, "<title>Admin Login</title>")
}

func TestLoginHandler_POST_Success(t *testing.T) {
	os.Setenv("ADMIN_USERNAME", "testuser")
	os.Setenv("ADMIN_PASSWORD", "testpass")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	h := setupTest(t)

	r := mux.NewRouter()
	r.HandleFunc("/admin/login", h.LoginHandler)

	ts := httptest.NewServer(mockCSRF(r))
	defer ts.Close()

	jar, err := cookiejar.New(nil)
	assert.NoError(t, err)
	client := &http.Client{Jar: jar}

	credentials := map[string]string{"username": "testuser", "password": "testpass"}
	body, err := json.Marshal(credentials)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", ts.URL+"/admin/login", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	postResp, err := client.Do(req)
	assert.NoError(t, err)
	defer postResp.Body.Close()

	assert.Equal(t, http.StatusOK, postResp.StatusCode)
	var respBody map[string]string
	err = json.NewDecoder(postResp.Body).Decode(&respBody)
	assert.NoError(t, err)
	assert.Equal(t, h.I18n.GetTranslation("en", "login_successful"), respBody["message"])
}

func TestLoginHandler_POST_InvalidCredentials(t *testing.T) {
	h := setupTest(t)

	credentials := map[string]string{"username": "wronguser", "password": "wrongpass"}
	body, err := json.Marshal(credentials)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/admin/login", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	h.LoginHandler(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "invalid_credentials"))
}

func TestLogoutHandler(t *testing.T) {
	h := setupTest(t)
	os.Setenv("ADMIN_USERNAME", "testuser")
	os.Setenv("ADMIN_PASSWORD", "testpass")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	// 1. Simulate Login to get a session cookie
	credentials := map[string]string{"username": "testuser", "password": "testpass"}
	body, err := json.Marshal(credentials)
	assert.NoError(t, err)

	loginReq, err := http.NewRequest("POST", "/admin/login", bytes.NewBuffer(body))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	ctx := context.WithValue(loginReq.Context(), csrf.TemplateTag, "dummy_token")
	loginReq = loginReq.WithContext(ctx)

	loginRr := httptest.NewRecorder()
	h.LoginHandler(loginRr, loginReq)
	assert.Equal(t, http.StatusOK, loginRr.Code)

	// Get the cookie from the login response
	loginCookies := loginRr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range loginCookies {
		if c.Name == "gemini-session" {
			sessionCookie = c
			break
		}
	}
	assert.NotNil(t, sessionCookie)

	// 2. Call LogoutHandler with the session cookie
	logoutReq, err := http.NewRequest("GET", "/admin/logout", nil)
	assert.NoError(t, err)
	logoutReq.AddCookie(sessionCookie)

	logoutRr := httptest.NewRecorder()
	h.LogoutHandler(logoutRr, logoutReq)

	// Assertions
	assert.Equal(t, http.StatusFound, logoutRr.Code)
	assert.Equal(t, "/admin/login", logoutRr.Header().Get("Location"))

	// Check that the session cookie is cleared
	logoutCookies := logoutRr.Result().Cookies()
	foundCookie := false
	for _, cookie := range logoutCookies {
		if cookie.Name == "gemini-session" {
			foundCookie = true
			assert.Equal(t, -1, cookie.MaxAge)
		}
	}
	assert.True(t, foundCookie)
}

type MockAuthService struct {
	AuthenticateFunc func(username, password string) bool
	LoginFunc        func(w http.ResponseWriter, r *http.Request) error
	LogoutFunc       func(w http.ResponseWriter, r *http.Request) error
	IsLoggedInFunc   func(r *http.Request) bool
	MiddlewareFunc   func(next http.Handler) http.Handler
}

func (m *MockAuthService) Authenticate(username, password string) bool {
	if m.AuthenticateFunc != nil {
		return m.AuthenticateFunc(username, password)
	}
	return false
}

func (m *MockAuthService) Login(w http.ResponseWriter, r *http.Request) error {
	if m.LoginFunc != nil {
		return m.LoginFunc(w, r)
	}
	return nil
}

func (m *MockAuthService) Logout(w http.ResponseWriter, r *http.Request) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(w, r)
	}
	return nil
}

func (m *MockAuthService) IsLoggedIn(r *http.Request) bool {
	if m.IsLoggedInFunc != nil {
		return m.IsLoggedInFunc(r)
	}
	return false
}

func (m *MockAuthService) Middleware(next http.Handler) http.Handler {
	if m.MiddlewareFunc != nil {
		return m.MiddlewareFunc(next)
	}
	return next
}

func TestLogoutHandler_Error(t *testing.T) {
	h := setupTest(t)

	// Create a mock auth service that returns an error on logout
	mockAuth := &MockAuthService{
		LogoutFunc: func(w http.ResponseWriter, r *http.Request) error {
			return fmt.Errorf("session save error")
		},
	}
	h.AuthService = mockAuth

	req, err := http.NewRequest("GET", "/admin/logout", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.LogoutHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "failed_to_log_out"))
}

func TestAdminRedirectHandler(t *testing.T) {
	h := setupTest(t)

	req, err := http.NewRequest("GET", "/admin", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.AdminRedirectHandler(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "/admin/dashboard", rr.Header().Get("Location"))
}

func TestDashboardHandler_Success(t *testing.T) {
	os.Setenv("ADMIN_USERNAME", "admin")
	os.Setenv("ADMIN_PASSWORD", "password")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	h := setupTest(t)

	r := mux.NewRouter()
	r.HandleFunc("/admin/dashboard", h.DashboardHandler)
	r.HandleFunc("/admin/login", h.LoginHandler)

	ts := httptest.NewServer(mockCSRF(r))
	defer ts.Close()

	jar, err := cookiejar.New(nil)
	assert.NoError(t, err)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	credentials := map[string]string{"username": "admin", "password": "password"}
	body, err := json.Marshal(credentials)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", ts.URL+"/admin/login", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	loginResp, err := client.Do(req)
	assert.NoError(t, err)
	defer loginResp.Body.Close()

	dashboardResp, err := client.Get(ts.URL + "/admin/dashboard")
	assert.NoError(t, err)
	defer dashboardResp.Body.Close()

	assert.Equal(t, http.StatusOK, dashboardResp.StatusCode)
	var dashboardBody []byte
	dashboardBody, err = io.ReadAll(dashboardResp.Body)
	assert.NoError(t, err)
	t.Logf("Dashboard Body: %s", string(dashboardBody))
	assert.Contains(t, string(dashboardBody), h.I18n.GetTranslation("en", "admin.dashboard_title"))
}

func TestMaintenanceMiddleware(t *testing.T) {
	h := setupTest(t)
	h.Cfg.Site.MaintenanceMode = true
	h.Cfg.Site.MaintenanceMessage = "Site is down for maintenance"

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := h.MaintenanceMiddleware(testHandler)

	// 1. Non-admin, not logged in -> Maintenance page
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "Site is down for maintenance")

	// 2. Admin route -> Bypassed
	req = httptest.NewRequest("GET", "/admin/dashboard", nil)
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())

	// 3. Logged in user -> Bypassed
	os.Setenv("ADMIN_USERNAME", "testuser")
	os.Setenv("ADMIN_PASSWORD", "testpass")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	// Create a router with all the necessary routes for the test
	r := mux.NewRouter()
	r.HandleFunc("/admin/login", h.LoginHandler)
	r.Handle("/", middleware) // Apply middleware to the root route

	ts := httptest.NewServer(mockCSRF(r))
	defer ts.Close()

	// Use a client with a cookie jar to handle sessions automatically
	jar, err := cookiejar.New(nil)
	assert.NoError(t, err)
	client := &http.Client{Jar: jar}

	// Step 1: Log in
	credentials := map[string]string{"username": "testuser", "password": "testpass"}
	body, err := json.Marshal(credentials)
	assert.NoError(t, err)

	loginReq, err := http.NewRequest("POST", ts.URL+"/admin/login", bytes.NewBuffer(body))
	assert.NoError(t, err)
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)

	// Step 2: Make a request to the page protected by the middleware
	bypassResp, err := client.Get(ts.URL + "/")
	assert.NoError(t, err)
	defer bypassResp.Body.Close()

	// Assert that the user bypassed the maintenance page
	assert.Equal(t, http.StatusOK, bypassResp.StatusCode)
	bypassBody, err := io.ReadAll(bypassResp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "OK", string(bypassBody))
}

func TestIndexHandler_Error(t *testing.T) {
	h := setupTest(t)
	h.Store = &MockErrorStore{}

	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.IndexHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "internal_server_error"))
}

func TestPagesListHandler_Error(t *testing.T) {
	h := setupTest(t)
	h.Store = &MockErrorStore{}

	req, err := http.NewRequest("GET", "/pages", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.PagesListHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "internal_server_error"))
}

func TestPagesListHandler_Success(t *testing.T) {
	h := setupTest(t)

	req, err := http.NewRequest("GET", "/pages", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.PagesListHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Pages")
	assert.Contains(t, rr.Body.String(), "Test Page Title")
}

func TestPageHandler_SiteConfigError(t *testing.T) {
	h := setupTest(t)

	// Create a custom mock store
	customStore := &CustomMockStore{}

	// Mock GetPageData to succeed
	customStore.GetPageDataFunc = func(name, lang, defaultLang string) (*models.Page, error) {
		return &models.Page{Name: "test-page", Content: models.PageTranslation{Title: "Test Page"}}, nil
	}

	// Mock GetSiteConfig to fail
	customStore.GetSiteConfigFunc = func(lang, defaultLang string) (*config.SiteConfig, error) {
		return nil, fmt.Errorf("database error")
	}

	h.Store = customStore

	pageName := "test-page"
	req, err := http.NewRequest("GET", fmt.Sprintf("/page/%s", pageName), nil)
	assert.NoError(t, err)

	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	h.PageHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "internal_server_error"))
}

func TestUpdatePageHandler_UpdatePageError(t *testing.T) {
	h := setupTest(t)

	customStore := &CustomMockStore{}
	customStore.GetPageDataFunc = func(name, lang, defaultLang string) (*models.Page, error) {
		return &models.Page{Name: "home"}, nil
	}
	customStore.UpdatePageFunc = func(page *models.Page) error {
		return fmt.Errorf("database error")
	}
	h.Store = customStore

	pageName := "home"
	updatedPayload := handler.PageUpdatePayload{Name: pageName, Title: "New Title", LanguageCode: "en"}
	body, err := json.Marshal(updatedPayload)
	assert.NoError(t, err)

	req, err := http.NewRequest("PUT", fmt.Sprintf("/pages/%s", pageName), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	h.UpdatePageHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestUpdatePageHandler_UpdatePageTranslationError(t *testing.T) {
	h := setupTest(t)

	customStore := &CustomMockStore{}
	customStore.GetPageDataFunc = func(name, lang, defaultLang string) (*models.Page, error) {
		return &models.Page{Name: "home"}, nil
	}
	customStore.UpdatePageFunc = func(page *models.Page) error {
		return nil
	}
	customStore.UpdatePageTranslationFunc = func(pageID uint, translation *models.PageTranslation) error {
		return fmt.Errorf("database error")
	}
	h.Store = customStore

	pageName := "home"
	updatedPayload := handler.PageUpdatePayload{Name: pageName, Title: "New Title", LanguageCode: "en"}
	body, err := json.Marshal(updatedPayload)
	assert.NoError(t, err)

	req, err := http.NewRequest("PUT", fmt.Sprintf("/pages/%s", pageName), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	h.UpdatePageHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestLoginHandler_POST_Success_CreateLoginLogError(t *testing.T) {
	os.Setenv("ADMIN_USERNAME", "testuser")
	os.Setenv("ADMIN_PASSWORD", "testpass")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	h := setupTest(t)

	customStore := &CustomMockStore{}
	customStore.CreateLoginLogFunc = func(log *models.LoginLog) error {
		return fmt.Errorf("database error")
	}
	h.Store = customStore

	r := mux.NewRouter()
	r.HandleFunc("/admin/login", h.LoginHandler)

	ts := httptest.NewServer(mockCSRF(r))
	defer ts.Close()

	jar, err := cookiejar.New(nil)
	assert.NoError(t, err)
	client := &http.Client{Jar: jar}

	credentials := map[string]string{"username": "testuser", "password": "testpass"}
	body, err := json.Marshal(credentials)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", ts.URL+"/admin/login", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	postResp, err := client.Do(req)
	assert.NoError(t, err)
	defer postResp.Body.Close()

	assert.Equal(t, http.StatusOK, postResp.StatusCode)
}

func TestLoginHandler_POST_InvalidCredentials_CreateLoginLogError(t *testing.T) {
	h := setupTest(t)

	customStore := &CustomMockStore{}
	customStore.CreateLoginLogFunc = func(log *models.LoginLog) error {
		return fmt.Errorf("database error")
	}
	h.Store = customStore

	credentials := map[string]string{"username": "wronguser", "password": "wrongpass"}
	body, err := json.Marshal(credentials)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/admin/login", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	h.LoginHandler(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRenderTemplate_TemplateNotFound(t *testing.T) {
	h := setupTest(t)
	h.Templates = make(map[string]*template.Template)

	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.IndexHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "internal_server_error"))
}

func TestRenderTemplate_ParseError(t *testing.T) {
	h := setupTest(t)
	h.DebugMode = true

	// Create a temporary invalid template file in the test's temp directory
	// This tests the case where DebugMode is on and templates are re-parsed at runtime.
	projectRoot := util.ProjectRoot("") // This will be the temp dir
	invalidTemplatePath := filepath.Join(projectRoot, "templates", "another_invalid.html")
	invalidTemplateContent := []byte("{{.Invalid")
	err := os.WriteFile(invalidTemplatePath, invalidTemplateContent, 0644)
	assert.NoError(t, err)

	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.IndexHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestPageHandler_NilPageData(t *testing.T) {
	h := setupTest(t)

	customStore := &CustomMockStore{}
	customStore.GetPageDataFunc = func(name, lang, defaultLang string) (*models.Page, error) {
		return nil, nil
	}
	customStore.GetSiteConfigFunc = func(lang, defaultLang string) (*config.SiteConfig, error) {
		return &config.SiteConfig{}, nil
	}
	h.Store = customStore

	pageName := "any-page"
	req, err := http.NewRequest("GET", fmt.Sprintf("/page/%s", pageName), nil)
	assert.NoError(t, err)

	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	h.PageHandler(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestAdminTemplatesView(t *testing.T) {
	h := setupTest(t)

	// Create request
	req, err := http.NewRequest("GET", "/admin/templates", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	// We need to wrap the handler with the mock CSRF middleware
	mockCSRF(http.HandlerFunc(h.AdminTemplatesView)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "base.html")
	assert.Contains(t, rr.Body.String(), "admin/admin_dashboard.html")
}

func TestAdminSettingsHandler_Success(t *testing.T) {
	h := setupTest(t)

	req, err := http.NewRequest("GET", "/admin/settings", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminSettingsHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Check for a key part of the settings page
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "admin.settings_title"))
	// Check that the site title from the DB is present
	assert.Contains(t, rr.Body.String(), "My Awesome Website")
}

func TestAdminSettingsHandler_DBError(t *testing.T) {
	h := setupTest(t)
	h.Store = &MockErrorStore{} // Mock store to return errors

	req, err := http.NewRequest("GET", "/admin/settings", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminSettingsHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestAdminSettingsHandler_GetAllPagesError(t *testing.T) {
	h := setupTest(t)

	customStore := &CustomMockStore{}
	customStore.GetSiteConfigFunc = func(lang, defaultLang string) (*config.SiteConfig, error) {
		return &config.SiteConfig{}, nil
	}
	customStore.GetAllPagesFunc = func(lang, defaultLang string) ([]models.Page, error) {
		return nil, fmt.Errorf("db error")
	}
	h.Store = customStore

	req, err := http.NewRequest("GET", "/admin/settings", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminSettingsHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestUpdateSettingsHandler_Success(t *testing.T) {
	h := setupTest(t)

	form := bytes.NewBufferString("siteTitle=New+Site+Title&defaultLanguage=en&editLang=en")

	req, err := http.NewRequest("POST", "/admin/settings", form)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.UpdateSettingsHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "admin.settings_saved_successfully"))

	// Verify the data was "saved" by checking the re-rendered form
	assert.Contains(t, rr.Body.String(), "New Site Title")
}

func TestUpdateSettingsHandler_ValidationError(t *testing.T) {
	h := setupTest(t)

	// Submit an empty siteTitle, which is a required field
	form := bytes.NewBufferString("siteTitle=&defaultLanguage=en&editLang=en")

	req, err := http.NewRequest("POST", "/admin/settings", form)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.UpdateSettingsHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "admin.settings.error.title_required"))
}

func TestUpdateSettingsHandler_InvalidGAID(t *testing.T) {
	h := setupTest(t)

	// Submit an invalid Google Analytics ID
	form := bytes.NewBufferString("siteTitle=Test&defaultLanguage=en&editLang=en&googleAnalyticsID=invalid-id")

	req, err := http.NewRequest("POST", "/admin/settings", form)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.UpdateSettingsHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "settings.error.invalid_ga_id"))
}

func TestUpdateSettingsHandler_InvalidNavJSON(t *testing.T) {
	h := setupTest(t)

	// Submit invalid navigation JSON
	form := bytes.NewBufferString("siteTitle=Test&defaultLanguage=en&editLang=en&navigationJson=invalid-json")

	req, err := http.NewRequest("POST", "/admin/settings", form)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.UpdateSettingsHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "settings.error.invalid_nav_json"))
}

func TestUpdateSettingsHandler_SaveCarouselError(t *testing.T) {
	h := setupTest(t)

	// Custom mock store to simulate a save failure for carousel settings
	customStore := &CustomMockStore{}
	customStore.SaveSiteConfigFunc = func(siteConfig *config.SiteConfig, lang string) error {
		return nil // Main config saves successfully
	}
	customStore.SaveSettingFunc = func(key, value, lang string) error {
		if key == "homepage_carousel_pages" {
			return fmt.Errorf("disk is full") // Specific error for carousel
		}
		return nil
	}
	// Mock the functions needed for re-rendering the template on success
	customStore.GetSiteConfigFunc = func(lang, defaultLang string) (*config.SiteConfig, error) {
		return &config.SiteConfig{DefaultLanguage: "en"}, nil
	}
	customStore.GetAllPagesFunc = func(lang, defaultLang string) ([]models.Page, error) {
		return []models.Page{}, nil
	}
	customStore.GetSettingValueFunc = func(key, lang string) (string, error) {
		return "[]", nil
	}
	h.Store = customStore

	form := bytes.NewBufferString("siteTitle=Test&defaultLanguage=en&editLang=en&homepageCarouselPagesJson=[]")

	req, err := http.NewRequest("POST", "/admin/settings", form)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.UpdateSettingsHandler)).ServeHTTP(rr, req)

	// The handler should still succeed because saving carousel settings is not a critical failure
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "admin.settings_saved_successfully"))
}

func TestUpdateSettingsHandler_SaveError(t *testing.T) {
	h := setupTest(t)

	// Custom mock store to simulate a save failure
	customStore := &CustomMockStore{}
	customStore.SaveSiteConfigFunc = func(siteConfig *config.SiteConfig, lang string) error {
		return fmt.Errorf("disk is full")
	}
	// Need to mock these as they are called when re-rendering the template on error
	customStore.GetAllPagesFunc = func(lang, defaultLang string) ([]models.Page, error) {
		return []models.Page{}, nil
	}
	customStore.GetSettingValueFunc = func(key, lang string) (string, error) {
		return "[]", nil
	}
	// Since the original store is a real DB, we need to replace it completely
	// We can't just set one function on the real store.
	// So we create a new handler with the mock store.
	h.Store = customStore

	form := bytes.NewBufferString("siteTitle=New+Site+Title&defaultLanguage=en&editLang=en")

	req, err := http.NewRequest("POST", "/admin/settings", form)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.UpdateSettingsHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "admin.settings_save_failed"))
}

func TestAdminTemplateEditView_Success(t *testing.T) {
	h := setupTest(t)

	// 1. Test with a template file
	req, err := http.NewRequest("GET", "/admin/templates/edit?file=base.html&type=template", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateEditView)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "<!DOCTYPE html>")

	// 2. Test with a static file
	// Create a dummy static file to test editing it
	staticDir := filepath.Join(util.ProjectRoot(""), "static", "css")
	os.MkdirAll(staticDir, 0755)
	dummyCSSPath := filepath.Join(staticDir, "test.css")
	err = os.WriteFile(dummyCSSPath, []byte("body {}"), 0644)
	assert.NoError(t, err)

	req, err = http.NewRequest("GET", "/admin/templates/edit?file=css/test.css&type=static", nil)
	assert.NoError(t, err)

	rr = httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateEditView)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "body {}")
}

func TestAdminTemplateEditView_Security(t *testing.T) {
	h := setupTest(t)

	// Attempt path traversal
	req, err := http.NewRequest("GET", "/admin/templates/edit?file=../config.yml&type=template", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateEditView)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAdminTemplateEditView_NotFound(t *testing.T) {
	h := setupTest(t)

	req, err := http.NewRequest("GET", "/admin/templates/edit?file=nonexistent.html&type=template", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateEditView)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestAdminTemplateEditView_BadRequest(t *testing.T) {
	h := setupTest(t)

	// 1. Missing params
	req, err := http.NewRequest("GET", "/admin/templates/edit", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateEditView)).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// 2. Trying to edit a directory
	req, err = http.NewRequest("GET", "/admin/templates/edit?file=css&type=static", nil)
	assert.NoError(t, err)
	rr = httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateEditView)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminTemplateUpdate_Success(t *testing.T) {
	h := setupTest(t)
	projectRoot := util.ProjectRoot("")
	fileName := "test_update.html"
	templatesDir := filepath.Join(projectRoot, "templates")
	os.MkdirAll(templatesDir, 0755)
	filePath := filepath.Join(templatesDir, fileName)

	// Create the file with initial content
	err := os.WriteFile(filePath, []byte("Initial content"), 0644)
	assert.NoError(t, err)

	// Verify initial content
	initialBytes, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, "Initial content", string(initialBytes))

	updatedContent := "Updated content"
	form := fmt.Sprintf("filePath=%s&fileType=template&fileContent=%s", fileName, updatedContent)

	req, err := http.NewRequest("POST", "/admin/templates/edit", strings.NewReader(form))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateUpdate)).ServeHTTP(rr, req)

	// Check for redirect
	assert.Equal(t, http.StatusFound, rr.Code)
	location, err := rr.Result().Location()
	assert.NoError(t, err)
	assert.Contains(t, location.String(), "messageType=success")

	// Check if file content was updated
	fileBytes, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, updatedContent, string(fileBytes))
}

func TestAdminTemplateUpdate_Security(t *testing.T) {
	h := setupTest(t)

	// Attempt to write outside the allowed directory
	form := "filePath=../evil.txt&fileType=template&fileContent=hacked"

	req, err := http.NewRequest("POST", "/admin/templates/edit", strings.NewReader(form))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateUpdate)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)

	// Verify the file was not created
	_, err = os.Stat(filepath.Join(util.ProjectRoot(""), "evil.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestAdminTemplateUpdate_BadRequest(t *testing.T) {
	h := setupTest(t)

	// Missing filePath
	form := "fileType=template&fileContent=content"

	req, err := http.NewRequest("POST", "/admin/templates/edit", strings.NewReader(form))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateUpdate)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminTemplatePreview_Success(t *testing.T) {
	h := setupTest(t)

	// Create a dummy template file to test previewing it
	templatesDir := filepath.Join(util.ProjectRoot(""), "templates")
	dummyTemplatePath := filepath.Join(templatesDir, "test_preview.html")
	err := os.WriteFile(dummyTemplatePath, []byte("{{define \"content\"}}Preview: {{.SiteConfig.Title}}{{end}}"), 0644)
	assert.NoError(t, err)

	req, err := http.NewRequest("GET", "/admin/templates/preview?file=test_preview.html", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplatePreview)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	// Check that the site title from the DB is rendered
	assert.Contains(t, rr.Body.String(), "Preview: My Awesome Website")
}

func TestAdminTemplatePreview_Security(t *testing.T) {
	h := setupTest(t)

	// Attempt path traversal
	req, err := http.NewRequest("GET", "/admin/templates/preview?file=../config.yml", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplatePreview)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestAdminTemplatePreview_BadRequest(t *testing.T) {
	h := setupTest(t)

	req, err := http.NewRequest("GET", "/admin/templates/preview", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplatePreview)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAdminTemplatePreview_ParseError(t *testing.T) {
	h := setupTest(t)

	req, err := http.NewRequest("GET", "/admin/templates/preview?file=invalid_preview.html", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplatePreview)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "Error parsing template for preview")
}

func TestAdminTemplatePreview_DBError(t *testing.T) {
	h := setupTest(t)
	h.Store = &MockErrorStore{} // Mock store to return errors

	req, err := http.NewRequest("GET", "/admin/templates/preview?file=db_error_preview.html", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplatePreview)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestAdminNewPageHandler_ShowForm_Success(t *testing.T) {
	h := setupTest(t)

	req, err := http.NewRequest("GET", "/admin/pages/new", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminNewPageHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "admin.new_page_title"))
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "admin.create_page_button"))
}

func TestAdminNewPageHandler_CreatePage_Success(t *testing.T) {
	h := setupTest(t)

	payload := map[string]interface{}{
		"Name":  "new-page-from-test",
		"Title": "New Page Title",
	}
	body, err := json.Marshal(payload)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/admin/pages/new", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminNewPageHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var createdPage models.Page
	err = json.NewDecoder(rr.Body).Decode(&createdPage)
	assert.NoError(t, err)
	assert.Equal(t, "new-page-from-test", createdPage.Name)
	assert.Equal(t, "New Page Title", createdPage.Content.Title)
}

func TestAdminNewPageHandler_CreatePage_ValidationError(t *testing.T) {
	h := setupTest(t)

	// Missing Name
	payload := map[string]interface{}{"Title": "New Page Title"}
	body, err := json.Marshal(payload)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/admin/pages/new", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminNewPageHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "page_name_title_required"))
}

func TestAdminNewPageHandler_CreatePage_Conflict(t *testing.T) {
	h := setupTest(t)

	// "home" page already exists from setupTest
	payload := map[string]interface{}{"Name": "home", "Title": "Home Title Again"}
	body, err := json.Marshal(payload)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/admin/pages/new", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminNewPageHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "page_exists"))
}

func TestDeletePageHandler_Success(t *testing.T) {
	h := setupTest(t)

	// The page "test-page" is created in setupTest
	pageName := "test-page"

	req, err := http.NewRequest("DELETE", "/admin/pages/delete/"+pageName, nil)
	assert.NoError(t, err)

	vars := map[string]string{"name": pageName}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.DeletePageHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify the page is actually deleted
	_, err = h.Store.GetPageData(pageName, "en", "en")
	assert.Error(t, err, "Expected an error when getting a deleted page")
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestDeletePageHandler_NotFound(t *testing.T) {
	h := setupTest(t)
	pageName := "non-existent-page"

	req, err := http.NewRequest("DELETE", "/admin/pages/delete/"+pageName, nil)
	assert.NoError(t, err)

	vars := map[string]string{"name": pageName}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.DeletePageHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "page_not_found"))
}

func TestDeletePageHandler_DBError(t *testing.T) {
	h := setupTest(t)
	h.Store = &MockErrorStore{} // This mock returns an error for DeletePage

	pageName := "any-page"
	req, err := http.NewRequest("DELETE", "/admin/pages/delete/"+pageName, nil)
	assert.NoError(t, err)

	vars := map[string]string{"name": pageName}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.DeletePageHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestPagesHandler_Success(t *testing.T) {
	h := setupTest(t)

	req, err := http.NewRequest("GET", "/admin/pages", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.PagesHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Pages")
	assert.Contains(t, rr.Body.String(), "Test Page")
}

func TestPagesHandler_Error(t *testing.T) {
	h := setupTest(t)
	h.Store = &MockErrorStore{}

	req, err := http.NewRequest("GET", "/admin/pages", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.PagesHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestAdminEditPageHandler_NewTranslation(t *testing.T) {
	h := setupTest(t)

	// Request to edit the 'home' page in Japanese ('ja'), which doesn't exist yet.
	// The handler should prepare a blank form for the new translation.
	req, err := http.NewRequest("GET", "/admin/pages/edit/home?lang=xx", nil)
	assert.NoError(t, err)

	vars := map[string]string{"name": "home"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminEditPageHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check that the form is for editing, not creating a new page.
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "admin.save_changes_button"))

	// IMPORTANT: Check that the title field is empty, because we are creating a new translation,
	// not editing an existing one. It should not fall back to the English title in the input field.
	assert.Contains(t, rr.Body.String(), `id="title" value=""`)
}

func TestAdminEditPageHandler_NotFound(t *testing.T) {
	h := setupTest(t)

	req, err := http.NewRequest("GET", "/admin/pages/edit/non-existent-page", nil)
	assert.NoError(t, err)

	vars := map[string]string{"name": "non-existent-page"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminEditPageHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestAdminEditPageHandler_Error(t *testing.T) {
	h := setupTest(t)
	h.Store = &MockErrorStore{}

	req, err := http.NewRequest("GET", "/admin/pages/edit/any-page", nil)
	assert.NoError(t, err)

	vars := map[string]string{"name": "any-page"}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminEditPageHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestUpdatePageHandler_EmptyTitle(t *testing.T) {
	h := setupTest(t)

	pageName := "home"
	// Create a payload with an empty title
	updatedPayload := handler.PageUpdatePayload{Name: pageName, Title: " ", LanguageCode: "en"}
	body, err := json.Marshal(updatedPayload)
	assert.NoError(t, err)

	req, err := http.NewRequest("PUT", fmt.Sprintf("/pages/%s", pageName), bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	vars := map[string]string{
		"name": pageName,
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	h.UpdatePageHandler(rr, req)

	// Assert that the request is bad because the title is empty
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "title_required"))
}

func TestImageUploadHandler_Success(t *testing.T) {
	h := setupTest(t)
	uploadDir := filepath.Join(util.ProjectRoot(""), "data", "uploads")

	// Ensure the parent 'data' directory exists, but clean up the 'uploads' dir after the test
	os.MkdirAll(filepath.Dir(uploadDir), 0755)
	t.Cleanup(func() {
		os.RemoveAll(uploadDir)
	})

	// Create a multipart form body
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("image", "test.jpg")
	assert.NoError(t, err)
	_, err = part.Write([]byte("fake-image-data"))
	assert.NoError(t, err)
	writer.Close()

	req, err := http.NewRequest("POST", "/admin/upload", &requestBody)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.ImageUploadHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["url"], "/uploads/")

	// Verify the file was created
	fileName := strings.TrimPrefix(resp["url"], "/uploads/")
	filePath := filepath.Join(uploadDir, fileName)
	_, err = os.Stat(filePath)
	assert.NoError(t, err, "Uploaded file should exist on disk")
}

func TestImageUploadHandler_NoFile(t *testing.T) {
	h := setupTest(t)

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	// Intentionally do not add a file part
	writer.Close()

	req, err := http.NewRequest("POST", "/admin/upload", &requestBody)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.ImageUploadHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "unable_to_get_image_from_form"))
}

func TestImageUploadHandler_DirectoryCreationError(t *testing.T) {
	h := setupTest(t)
	dataDir := filepath.Join(util.ProjectRoot(""), "data")
	uploadPath := filepath.Join(dataDir, "uploads")

	// Remove the directory created by setupTest
	err := os.RemoveAll(uploadPath)
	assert.NoError(t, err)

	// Create a file where the directory should be, to cause the handler to fail
	err = os.WriteFile(uploadPath, []byte("this is a file"), 0644)
	assert.NoError(t, err)

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("image", "test.jpg")
	assert.NoError(t, err)
	_, err = part.Write([]byte("fake-image-data"))
	assert.NoError(t, err)
	writer.Close()

	req, err := http.NewRequest("POST", "/admin/upload", &requestBody)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.ImageUploadHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "unable_to_create_upload_directory"))
}

func TestTranslateHandler_Success(t *testing.T) {
	h := setupTest(t)
	payload := handler.TranslateRequest{
		SourceLanguage: "en",
		TargetLanguage: "ja",
		Content:        "Hello World",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/admin/api/translate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.TranslateHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp handler.TranslateResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Equal(t, "translated: Hello World", resp.TranslatedText)
}

func TestTranslateHandler_FetchPageContent(t *testing.T) {
	h := setupTest(t)
	// "home" page with title "Original Home Title" is created in setupTest
	payload := handler.TranslateRequest{
		PageName:       "home",
		Field:          "title",
		SourceLanguage: "en",
		TargetLanguage: "ja",
		Content:        "", // Intentionally empty to trigger fetch
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/admin/api/translate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.TranslateHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp handler.TranslateResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Equal(t, "translated: Innovatech - AI & IT Solutions", resp.TranslatedText)
}

func TestTranslateHandler_FetchPageContent_InvalidField(t *testing.T) {
	h := setupTest(t)
	payload := handler.TranslateRequest{
		PageName:       "home",
		Field:          "invalid-field", // This field does not exist
		SourceLanguage: "en",
		TargetLanguage: "ja",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/admin/api/translate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.TranslateHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "invalid_field_for_translation"))
}

func TestTranslateHandler_InvalidBody(t *testing.T) {
	h := setupTest(t)
	req, _ := http.NewRequest("POST", "/admin/api/translate", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.TranslateHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTranslateHandler_FetchContentNotFound(t *testing.T) {
	h := setupTest(t)
	payload := handler.TranslateRequest{
		PageName:       "non-existent-page",
		Field:          "title",
		SourceLanguage: "en",
		TargetLanguage: "ja",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/admin/api/translate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.TranslateHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "source_content_not_found"))
}

func TestTranslateHandler_TranslationAPIFailure(t *testing.T) {
	h := setupTest(t)
	// Configure the mock to return an error
	h.API_Translator = &MockAPITranslator{
		TranslateTextFunc: func(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
			return "", fmt.Errorf("API limit reached")
		},
	}

	payload := handler.TranslateRequest{
		SourceLanguage: "en",
		TargetLanguage: "ja",
		Content:        "Hello World",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/admin/api/translate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.TranslateHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "translation_failed"))
}

func TestTranslateHandler_FetchSettingContent(t *testing.T) {
	h := setupTest(t)
	// "site_title" with value "Test Title" is created in setupTest
	payload := handler.TranslateRequest{
		Field:          "site_title",
		SourceLanguage: "en",
		TargetLanguage: "ja",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/admin/api/translate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.TranslateHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp handler.TranslateResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Equal(t, "translated: My Awesome Website", resp.TranslatedText)
}

func TestTranslateHandler_FetchContent_NoContext(t *testing.T) {
	h := setupTest(t)
	// Content is empty, and no PageName or Field is provided
	payload := handler.TranslateRequest{
		SourceLanguage: "en",
		TargetLanguage: "ja",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/admin/api/translate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.TranslateHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestLoginHandler_GET_AlreadyLoggedIn(t *testing.T) {
	os.Setenv("ADMIN_USERNAME", "testuser")
	os.Setenv("ADMIN_PASSWORD", "testpass")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	h := setupTest(t)

	r := mux.NewRouter()
	r.HandleFunc("/admin/login", h.LoginHandler)
	r.HandleFunc("/admin/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("dashboard"))
	})

	ts := httptest.NewServer(mockCSRF(r))
	defer ts.Close()

	jar, err := cookiejar.New(nil)
	assert.NoError(t, err)
	client := &http.Client{
		Jar: jar,
		// Stop the client from following redirects automatically
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Step 1: Log in to establish a session
	credentials := map[string]string{"username": "testuser", "password": "testpass"}
	body, err := json.Marshal(credentials)
	assert.NoError(t, err)

	loginResp, err := client.Post(ts.URL+"/admin/login", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)
	loginResp.Body.Close()

	// Step 2: Try to access the login page again while logged in
	getResp, err := client.Get(ts.URL + "/admin/login")
	assert.NoError(t, err)
	defer getResp.Body.Close()

	// Assert that the user is redirected to the dashboard
	assert.Equal(t, http.StatusFound, getResp.StatusCode)
	location, err := getResp.Location()
	assert.NoError(t, err)
	assert.Equal(t, "/admin/dashboard", location.Path)
}

func TestDashboardHandler_PartialError(t *testing.T) {
	os.Setenv("ADMIN_USERNAME", "admin")
	os.Setenv("ADMIN_PASSWORD", "password")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	h := setupTest(t)

	// Create a custom mock store that fails on GetPageCount but succeeds on others
	customStore := &CustomMockStore{
		GetPageCountFunc: func() (int64, error) {
			return 0, fmt.Errorf("mock db error getting page count")
		},
		GetRecentPagesFunc: func(limit int, lang, defaultLang string) ([]models.Page, error) {
			// Return some data to ensure the template renders something
			return []models.Page{{Name: "recent-page", Content: models.PageTranslation{Title: "Recent Page"}}}, nil
		},
		GetRecentLoginLogsFunc: func(limit int) ([]models.LoginLog, error) {
			return []models.LoginLog{}, nil
		},
	}
	h.Store = customStore

	// Simulate being logged in
	loginRr := httptest.NewRecorder()
	loginReq, _ := http.NewRequest("GET", "/", nil)
	err := h.AuthService.Login(loginRr, loginReq)
	assert.NoError(t, err)
	sessionCookie := loginRr.Result().Cookies()[0]

	// Make request to the dashboard
	req, err := http.NewRequest("GET", "/admin/dashboard", nil)
	assert.NoError(t, err)
	req.AddCookie(sessionCookie)

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.DashboardHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Dashboard should render successfully even if some data calls fail")
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "admin.recent_pages"), "Dashboard should still contain content from successful data calls")
}

func TestGetLanguage_InvalidCookie(t *testing.T) {
	h := setupTest(t)

	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	// Set an invalid language cookie
	req.AddCookie(&http.Cookie{Name: "lang", Value: "invalid"})
	req.Header.Set("Accept-Language", "ja")

	rr := httptest.NewRecorder()
	h.IndexHandler(rr, req)

	// Check that the language falls back to the Accept-Language header
	assert.Contains(t, rr.Body.String(), "ホーム")
}

func TestUpdateSettingsHandler_InvalidLanguage(t *testing.T) {
	h := setupTest(t)

	form := bytes.NewBufferString("siteTitle=New+Site+Title&defaultLanguage=invalid&editLang=en")

	req, err := http.NewRequest("POST", "/admin/settings", form)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.UpdateSettingsHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid default language code")
}

func TestAdminNewPageHandler_CreatePage_DBError(t *testing.T) {
	h := setupTest(t)
	h.Store = &MockErrorStore{}

	payload := map[string]interface{}{
		"Name":  "new-page-from-test",
		"Title": "New Page Title",
	}
	body, err := json.Marshal(payload)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/admin/pages/new", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminNewPageHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestImageUploadHandler_RandReadError(t *testing.T) {
	h := setupTest(t)

	// Create a multipart form body
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("image", "test.jpg")
	assert.NoError(t, err)
	_, err = part.Write([]byte("fake-image-data"))
	assert.NoError(t, err)
	writer.Close()

	req, err := http.NewRequest("POST", "/admin/upload", &requestBody)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Mock rand.Read to return an error
	originalRandRead := handler.RandRead
	handler.RandRead = func(b []byte) (n int, err error) {
		return 0, fmt.Errorf("rand read error")
	}
	defer func() { handler.RandRead = originalRandRead }()

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.ImageUploadHandler)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), h.I18n.GetTranslation("en", "failed_to_generate_random_filename"))
}

func TestAdminTemplateUpdate_StatError(t *testing.T) {
	h := setupTest(t)

	form := "filePath=any.html&fileType=template&fileContent=content"
	req, err := http.NewRequest("POST", "/admin/templates/edit", strings.NewReader(form))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Mock os.Stat to return an error
	originalOsStat := handler.OsStat
	handler.OsStat = func(name string) (os.FileInfo, error) {
		return nil, fmt.Errorf("stat error")
	}
	defer func() { handler.OsStat = originalOsStat }()

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateUpdate)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestAdminTemplateUpdate_WriteToDirectory(t *testing.T) {
	h := setupTest(t)
	projectRoot := util.ProjectRoot("")
	dirName := "test_dir"
	templatesDir := filepath.Join(projectRoot, "templates")
	os.MkdirAll(filepath.Join(templatesDir, dirName), 0755)

	form := fmt.Sprintf("filePath=%s&fileType=template&fileContent=content", dirName)

	req, err := http.NewRequest("POST", "/admin/templates/edit", strings.NewReader(form))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateUpdate)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Cannot write to a directory")
}

func TestAdminTemplateUpdate_WriteFileError(t *testing.T) {
	h := setupTest(t)
	projectRoot := util.ProjectRoot("")
	fileName := "test_write_error.html"
	templatesDir := filepath.Join(projectRoot, "templates")
	os.MkdirAll(templatesDir, 0755)
	filePath := filepath.Join(templatesDir, fileName)

	// Create the file as read-only to cause a write error
	err := os.WriteFile(filePath, []byte("Initial content"), 0444)
	assert.NoError(t, err)

	form := fmt.Sprintf("filePath=%s&fileType=template&fileContent=new content", fileName)

	req, err := http.NewRequest("POST", "/admin/templates/edit", strings.NewReader(form))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	mockCSRF(http.HandlerFunc(h.AdminTemplateUpdate)).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code) // It redirects on failure
	location, err := rr.Result().Location()
	assert.NoError(t, err)
	assert.Contains(t, location.String(), "messageType=error")
}

func TestMaintenanceMiddleware_TemplateParseError(t *testing.T) {
	h := setupTest(t)
	h.Cfg.Site.MaintenanceMode = true

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := h.MaintenanceMiddleware(testHandler)

	// Create an invalid maintenance template
	projectRoot := util.ProjectRoot("")
	invalidTemplatePath := filepath.Join(projectRoot, "templates", "maintenance.html")
	invalidTemplateContent := []byte("{{.Invalid")
	err := os.WriteFile(invalidTemplatePath, invalidTemplateContent, 0644)
	assert.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}
