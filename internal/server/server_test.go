package server

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"gemini-demo/internal/auth"
	"gemini-demo/internal/config"
	"gemini-demo/internal/i18n"
	"gemini-demo/internal/logger"
	"gemini-demo/internal/models"
	"gemini-demo/internal/testutil"
	"gemini-demo/internal/translator"
	"gemini-demo/internal/util"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockAPITranslator is a mock implementation of the translator.Translator interface for testing.
type MockAPITranslator struct{}

func (m *MockAPITranslator) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	// For testing, just return the original text with language codes
	return fmt.Sprintf("%s (translated from %s to %s)", text, sourceLang, targetLang), nil
}

func (m *MockAPITranslator) Close() error {
	return nil
}

var _ translator.Translator = (*MockAPITranslator)(nil)

// debugLog is a dummy function for testing to prevent nil pointer dereference
func debugLog(format string, v ...interface{}) {
	// Do nothing or log to t.Logf for debugging tests
	// t.Logf(format, v...)
}

// setupServerTest initializes a full server instance for integration testing.
func setupServerTest(t *testing.T) *http.Server {
	tmpDir, cleanup, err := testutil.SetupTestEnv(t)
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	// Overwrite seed_data.json to ensure tests don't fail on malformed test data.
	seedDataPath := filepath.Join(tmpDir, "data", "seed_data.json")
	if err := os.WriteFile(seedDataPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write dummy seed_data.json: %v", err)
	}
	
	originalWD, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(tmpDir)
	assert.NoError(t, err)

	t.Cleanup(func() {
		os.Chdir(originalWD)
		cleanup()
	})

	templateDir := filepath.Join(util.ProjectRoot(""), "templates")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		t.Fatalf("templates directory does not exist in test root: %s", templateDir)
	}

	// Use assert for cleaner test failures
	assert := assert.New(t)

	cfg := &config.Config{ // Minimal config for routing
		Auth: config.AuthConfig{
			SessionKey: "test-secret-key-for-sessions-32",
		},
		Static: config.StaticConfig{
			URLPrefix: "/static/",
			Dir:       "static",
		},
		DebugMode: false, // Set debug mode for tests
		Routes: []config.Route{
			{Path: "/", Handler: "IndexHandler", Methods: []string{"GET"}, AuthRequired: false},
			{Path: "/about", Handler: "AboutHandler", Methods: []string{"GET"}, AuthRequired: false},
			{Path: "/admin/dashboard", Handler: "DashboardHandler", Methods: []string{"GET"}, AuthRequired: true},
			{Path: "/admin/login", Handler: "LoginHandler", Methods: []string{"GET", "POST"}, AuthRequired: false}, // Keep for login simulation
			{Path: "/nonexistent", Handler: "NonExistentHandler", Methods: []string{"GET"}, AuthRequired: false},
		},
	}

	// Use in-memory SQLite for cleaner tests
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(err)

	sqlDB, err := db.DB()
	assert.NoError(err)
	t.Cleanup(func() {
		sqlDB.Close()
	})

	// Migrate and seed the in-memory database
	err = models.AutoMigrateAndSeed(db)
	assert.NoError(err)

	// Initialize i18n translator for testing
	i18nBasePath := filepath.Join(util.ProjectRoot(""), "data", "i18n")
	i18nTranslator := i18n.NewTranslator(i18nBasePath, "en") // "en" as default language
	err = i18nTranslator.LoadTranslations()
	assert.NoError(err)

	// Parse templates for testing
	templatesMap, err := util.ParseTemplates(i18nTranslator, util.ProjectRoot(""))
	assert.NoError(err)

	// Create a mock CSRF middleware that just passes through
	mockCSRFMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	errorLogger := logger.NewInMemoryLogCollector(100)
	startTime := time.Now()
	return New(cfg, db, templatesMap, mockCSRFMiddleware, i18nTranslator, &MockAPITranslator{}, debugLog, errorLogger, startTime)
}

func TestNew(t *testing.T) {
	srv := setupServerTest(t)

	t.Run("serves static files", func(t *testing.T) {
		// The test environment is set up by setupServerTest, which copies the static directory.
		// We just need to verify a file can be served.
		// Create a dummy file in the temporary static directory to test against.
		staticDir := filepath.Join(util.ProjectRoot(""), "static")
		dummyFilePath := filepath.Join(staticDir, "test.txt")
		dummyContent := "Hello from static file!"
		err := os.WriteFile(dummyFilePath, []byte(dummyContent), 0644)
		if err != nil {
			t.Fatalf("failed to write dummy static file: %v", err)
		}

		// Log the contents of the static directory
		files, err := ioutil.ReadDir(staticDir)
		if err != nil {
			t.Fatalf("failed to read static dir: %v", err)
		}
		var fileNames []string
		for _, f := range files {
			fileNames = append(fileNames, f.Name())
		}
		t.Logf("Static dir contents: %v", fileNames)

		req, err := http.NewRequest("GET", "/static/test.txt", nil)
		if err != nil {
			t.Fatalf("could not create request: %v", err)
		}
		rr := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code for static file: got %v want %v",
				status, http.StatusOK)
		}

		if rr.Body.String() != dummyContent {
			t.Errorf("static file content mismatch: got %q want %q",
				rr.Body.String(), dummyContent)
		}
	})



	// Test unauthenticated access to /admin/dashboard
	t.Run("redirects unauthenticated access to /admin/dashboard", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/admin/dashboard", nil)
		if err != nil {
			t.Fatalf("could not create request: %v", err)
		}
		rr := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusFound {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusFound)
		}
		if location := rr.Header().Get("Location"); location != "/admin/login" {
			t.Errorf("handler redirected to wrong location: got %v want %v", location, "/admin/login")
		}
	})

	// Test authenticated access to /admin/dashboard
	t.Run("allows authenticated access to /admin/dashboard", func(t *testing.T) {
		assert := assert.New(t)
		authService := auth.NewAuthService("test-secret-key-for-sessions-32")

		// Simulate being logged in by creating a valid session cookie
		loginRr := httptest.NewRecorder()
		loginReq, _ := http.NewRequest("GET", "/", nil) // Dummy request to get a session
		err := authService.Login(loginRr, loginReq)
		assert.NoError(err)
		sessionCookie := loginRr.Result().Cookies()[0]

		// Make request to the protected dashboard route with the cookie
		req, err := http.NewRequest("GET", "/admin/dashboard", nil)
		assert.NoError(err)
		req.AddCookie(sessionCookie)

		rr := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rr, req)

		assert.Equal(http.StatusOK, rr.Code)
		assert.Contains(rr.Body.String(), "Admin Dashboard")
	})

	// Test for a route with a handler name that does not exist in the handlers map
	t.Run("handles nonexistent handler name in config", func(t *testing.T) {
		assert := assert.New(t)
		req, err := http.NewRequest("GET", "/nonexistent", nil)
		assert.NoError(err)
		rr := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rr, req)

		assert.Equal(http.StatusNotFound, rr.Code)
	})
}

func TestNew_PanicOnNilConfig(t *testing.T) {
	// Expect a panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected New to panic, but it did not")
		}
	}()

	// Call New, which should panic
	New(nil, nil, nil, nil, nil, nil, nil, nil, time.Time{}) // Pass nil for cfg, db, tmpl, csrfMiddleware, and translator
}

func TestNew_LoginSimulation(t *testing.T) {
	os.Setenv("ADMIN_USERNAME", "admin")
	os.Setenv("ADMIN_PASSWORD", "password")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	srv := setupServerTest(t)
	ts := httptest.NewServer(srv.Handler)
	defer ts.Close()

	// This test ensures the login handler is correctly wired up in the server
	resp, err := http.Post(ts.URL+"/admin/login", "application/json", bytes.NewBufferString(`{"username":"admin","password":"password"}`))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}