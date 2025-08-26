package server_test

import (
	"fmt"
	"html/template"
	"gemini-demo/internal/auth"
	"gemini-demo/internal/database"
	"gemini-demo/internal/handler"
	"gemini-demo/internal/models"
	"gemini-demo/internal/server"
	"gemini-demo/internal/util"
	"gemini-demo/tests/testutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// parseTemplates walks the templates directory and parses all .html files.
func parseTemplates() (*template.Template, error) {
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

	tmpl, err := template.ParseFiles(templateFiles...)
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %w", err)
	}
	return tmpl, nil
}

func TestNew(t *testing.T) {
	// Set up the database for testing
	testutil.SetupViper()
	vp := viper.GetViper()
	vp.Set("auth.session_key", "test-secret-key-for-sessions-32")
	vp.Set("static.url_prefix", "/static/")
	vp.Set("static.dir", "static")

	// Configure routes for testing
	vp.Set("routes", []map[string]interface{}{
		{"path": "/", "handler": "IndexHandler", "methods": []string{"GET"}, "auth_required": false},
		{"path": "/about", "handler": "AboutHandler", "methods": []string{"GET"}, "auth_required": false},
		{"path": "/admin/dashboard", "handler": "DashboardHandler", "methods": []string{"GET"}, "auth_required": true},
		{"path": "/admin/login", "handler": "LoginHandler", "methods": []string{"GET"}, "auth_required": false},
		{"path": "/admin/redirect", "handler": "AdminRedirectHandler", "methods": []string{"GET"}, "auth_required": true},
		{"path": "/nonexistent", "handler": "NonExistentHandler", "methods": []string{"GET"}, "auth_required": false},
	})

	db, sqlDB, err := database.InitDB()
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

	// Parse templates for testing
	tmpl, err := parseTemplates()
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	// Create a mock CSRF middleware that just passes through
	mockCSRFMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	srv := server.New(db, tmpl, mockCSRFMiddleware)


	t.Run("serves the hello handler at the root", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatalf("could not create request: %v", err)
		}
		rr := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		// Check if the response body contains the data from the database
		expectedTitle := "<h1>构建未来, <span>智能驱动</span></h1>"
		if !strings.Contains(rr.Body.String(), expectedTitle) {
			t.Errorf("handler returned unexpected body: got %v want to contain %v",
				rr.Body.String(), expectedTitle)
		}

		expectedMessage := "<p>我们提供尖端的IT基础架构和人工智能解决方案，帮助您的企业在数字化浪潮中保持领先。</p>"
		if !strings.Contains(rr.Body.String(), expectedMessage) {
			t.Errorf("handler returned unexpected body: got %v want to contain %v",
				rr.Body.String(), expectedMessage)
		}
	})

	t.Run("serves static files", func(t *testing.T) {
		// Create a dummy static file
		staticDir := viper.GetString("static.dir")
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

		expected := `This is the about page.`
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
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
		authService := auth.NewAuthServiceWithViper(viper.GetViper())
		h := &handler.Handler{DB: db, AuthService: authService}

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

func TestNew_UnmarshalKeyError(t *testing.T) {
	// Save current viper settings and restore them after the test
	originalRoutes := viper.Get("routes")
	defer func() {
		viper.Set("routes", originalRoutes)
	}()

	// Intentionally set an invalid value for "routes" to cause UnmarshalKey to fail
	viper.Set("routes", "invalid_routes_value")

	// Expect a panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected New to panic, but it did not")
		}
	}()

	// Call New, which should panic
	server.New(nil, nil, nil) // Pass nil for db, tmpl, and csrfMiddleware
}
