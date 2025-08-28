package auth_test

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gemini-demo/internal/auth"
	"gemini-demo/internal/handler"
	"gemini-demo/internal/i18n"
)

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
		os.Unsetenv("ADMIN_USERNAME")
		os.Unsetenv("ADMIN_PASSWORD")
		if !authService.Authenticate("admin", "password") {
			t.Error("Authenticate failed for valid credentials with default values")
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

	// Test IsLoggedIn with no session cookie
	t.Run("IsLoggedIn returns false with no session cookie", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/", nil)
		if authService.IsLoggedIn(req) {
			t.Error("IsLoggedIn returned true with no session cookie")
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

        // Get current working directory
        wd, err := os.Getwd()
        if err != nil {
            t.Fatalf("Failed to get current working directory: %v", err)
        }

        // Construct absolute path to i18n directory
        i18nPath := filepath.Join(wd, "..", "..", "..", "data", "i18n")

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
            AuthService: authService,
            Templates:   tmpl,
            Translator:  translator,
            DebugLog:    debugLog,
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
