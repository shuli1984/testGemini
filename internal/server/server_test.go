package server

import (
	
	"fmt"
	"html/template"
	"gemini-demo/internal/auth"
	"gemini-demo/internal/config"
	"gemini-demo/internal/database"
	"gemini-demo/internal/handler"
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

// parseTemplates walks the templates directory and parses all .html files.
func parseTemplates(translator *i18n.Translator) (*template.Template, error) {
	projectRoot := util.ProjectRoot("") // Using util.ProjectRoot
	var templateFiles []string
	err := filepath.Walk(filepath.Join(projectRoot, "templates"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			templateFiles = append(templateFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking templates directory: %w", err)
	}

	if len(templateFiles) == 0 {
		return nil, fmt.Errorf("no HTML templates found in %s", filepath.Join(projectRoot, "templates"))
	}

	// Create a FuncMap for templates
	funcMap := template.FuncMap{
		"T": func(lang, key string) string {
			return translator.GetTranslation(lang, key)
		},
		"hasPrefix": strings.HasPrefix,
	}

	tmpl := template.New("main").Funcs(funcMap)
	tmpl, err = tmpl.ParseFiles(templateFiles...)
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %w", err)
	}
	return tmpl, nil
}

// debugLog is a dummy function for testing to prevent nil pointer dereference
func debugLog(format string, v ...interface{}) {
	// Do nothing or log to t.Logf for debugging tests
	// t.Logf(format, v...)
}

func TestNew(t *testing.T) {
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
			{Path: "/admin/login", Handler: "LoginHandler", Methods: []string{"GET"}, AuthRequired: false},
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
	tmpl, err := parseTemplates(translator)
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	// Create a mock CSRF middleware that just passes through
	mockCSRFMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	srv := New(cfg, db, tmpl, mockCSRFMiddleware, translator, debugLog, false)

	

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
		// Simulate a logged-in session
		// We need a handler instance to call AuthService.Login
		authService := auth.NewAuthService(cfg.Auth.SessionKey)
		h := &handler.Handler{Store: &models.DBStore{DB: db}, AuthService: authService, Translator: translator, DebugLog: debugLog}

		// Create a dummy request for Login to set the cookie
		loginReq, err := http.NewRequest("POST", "/admin/login", strings.NewReader(`{"username":"admin","password":"password"}`))
		if err != nil {
			t.Fatalf("could not create login request: %v", err)
		}
		loginRr := httptest.NewRecorder()
		h.LoginHandler(loginRr, loginReq)

		if loginRr.Code != http.StatusOK {
			t.Fatalf("failed to simulate login: status %d, body %s", loginRr.Code, loginRr.Body.String())
		}

		// Get the session cookie from the login response
		var sessionCookie *http.Cookie
		for _, cookie := range loginRr.Result().Cookies() {
			if cookie.Name == "gemini-session" {
				sessionCookie = cookie
				break
			}
		}
		if sessionCookie == nil {
			t.Fatalf("session cookie not found after simulated login")
		}

		// Now make the actual request to /admin/dashboard with the session cookie
		req, err := http.NewRequest("GET", "/admin/dashboard", nil)
		if err != nil {
			t.Fatalf("could not create dashboard request: %v", err)
		}
		req.AddCookie(sessionCookie)

		rr := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code for authenticated access: got %v want %v",
				status, http.StatusOK)
		}
		// Further checks can be added here to verify dashboard content
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
