package server

import (
	"gemini-demo/internal/config"
	"gemini-demo/internal/database"
	"gemini-demo/internal/i18n" // Added for i18n
	"gemini-demo/internal/models"
	"gemini-demo/internal/util"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)



// debugLog is a dummy function for testing to prevent nil pointer dereference
func debugLog(format string, v ...interface{}) {
	// Do nothing or log to t.Logf for debugging tests
	// t.Logf(format, v...)
}

func TestNew(t *testing.T) {
	os.Setenv("ADMIN_USERNAME", "admin")
	os.Setenv("ADMIN_PASSWORD", "password")
	// Set up the database for testing
	
	cfg := &config.Config{
		Auth: config.AuthConfig{
			SessionKey: "test-secret-key-for-sessions-32",
		},
		Static: config.StaticConfig{
			URLPrefix: "/static/",
			Dir:       "static",
		},
		Routes: []config.Route{
			{Path: "/", Handler: "IndexHandler", Methods: []string{"GET"}, AuthRequired: false},
			{Path: "/about", Handler: "AboutHandler", Methods: []string{"GET"}, AuthRequired: false},
			{Path: "/admin/dashboard", Handler: "DashboardHandler", Methods: []string{"GET"}, AuthRequired: true},
			{Path: "/admin/login", Handler: "LoginHandler", Methods: []string{"GET", "POST"}, AuthRequired: false},
			{Path: "/admin/redirect", Handler: "AdminRedirectHandler", Methods: []string{"GET"}, AuthRequired: true},
			{Path: "/nonexistent", Handler: "NonExistentHandler", Methods: []string{"GET"}, AuthRequired: false},
		},
	}

	db, sqlDB, err := database.InitDB("sqlite", "./gemini.db")
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}
	defer func() {
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()
	models.AutoMigrateAndSeed(db)
	defer os.Remove("./gemini.db")

	// Initialize i18n translator for testing
	i18nBasePath := filepath.Join(util.ProjectRoot(""), "data", "i18n")
	translator := i18n.NewTranslator(i18nBasePath, "en") // "en" as default language
	if err := translator.LoadTranslations(); err != nil {
		t.Fatalf("Failed to load translations for test: %v", err)
	}

	// Parse templates for testing
	templatesMap, err := util.ParseTemplates(translator)
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	// Create a mock CSRF middleware that just passes through
	mockCSRFMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	srv := New(cfg, db, templatesMap, mockCSRFMiddleware, translator, debugLog, false) // Use the map directly

	

	t.Run("serves static files", func(t *testing.T) {
		// Create a dummy static file
		staticDir := cfg.Static.Dir
		if err := os.MkdirAll(staticDir, 0755); err != nil {
			t.Fatalf("failed to create static dir: %v", err)
		}
		defer os.RemoveAll(staticDir)

		dummyFilePath := filepath.Join(staticDir, "test.txt")
		dummyContent := "Hello from static file!"
		if err := os.WriteFile(dummyFilePath, []byte(dummyContent), 0644); err != nil {
			t.Fatalf("failed to write dummy static file: %v", err)
		}

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

	t.Run("serves the about handler at /about", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/about", nil)
		if err != nil {
			t.Fatalf("could not create request: %v", err)
		}
		rr := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		// Check for key content in the rendered HTML
		expectedContent := "about_hero_title"
		if !strings.Contains(rr.Body.String(), expectedContent) {
			t.Errorf("handler returned unexpected body: expected to contain %q, got %q",
				expectedContent, rr.Body.String())
		}
		expectedContent = "our_story_content"
		if !strings.Contains(rr.Body.String(), expectedContent) {
			t.Errorf("handler returned unexpected body: expected to contain %q, got %q",
				expectedContent, rr.Body.String())
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
		// Create a new server for this test
		ts := httptest.NewServer(srv.Handler)
		defer ts.Close()

		// Create a client with a cookie jar to store the session cookie
		client := &http.Client{}

		// Simulate login
		loginURL := ts.URL + "/admin/login"
		loginBody := strings.NewReader(`{"username":"admin","password":"password"}`)
		loginReq, err := http.NewRequest("POST", loginURL, loginBody)
		if err != nil {
			t.Fatalf("could not create login request: %v", err)
		}
		loginReq.Header.Set("Content-Type", "application/json")

		loginResp, err := client.Do(loginReq)
		if err != nil {
			t.Fatalf("login request failed: %v", err)
		}
		defer loginResp.Body.Close()

		if loginResp.StatusCode != http.StatusOK {
			t.Fatalf("failed to simulate login: status %d", loginResp.StatusCode)
		}

		// Now make the actual request to /admin/dashboard with the session cookie
		dashboardURL := ts.URL + "/admin/dashboard"
		req, err := http.NewRequest("GET", dashboardURL, nil)
		if err != nil {
			t.Fatalf("could not create dashboard request: %v", err)
		}

		// Add cookies from the login response to the new request
		for _, cookie := range loginResp.Cookies() {
			req.AddCookie(cookie)
		}

		dashboardResp, err := client.Do(req)
		if err != nil {
			t.Fatalf("dashboard request failed: %v", err)
		}
		defer dashboardResp.Body.Close()

		if status := dashboardResp.StatusCode; status != http.StatusOK {
			t.Errorf("handler returned wrong status code for authenticated access: got %v want %v",
				status, http.StatusOK)
		}
	})

	// Test for a route with a handler name that does not exist in the handlers map
	t.Run("handles nonexistent handler name in config", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/nonexistent", nil)
		if err != nil {
			t.Fatalf("could not create request: %v", err)
		}
		rr := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code for nonexistent handler: got %v want %v",
				status, http.StatusNotFound)
		}
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
	New(nil, nil, nil, nil, nil, nil, false) // Pass nil for cfg, db, tmpl, csrfMiddleware, and translator
}
