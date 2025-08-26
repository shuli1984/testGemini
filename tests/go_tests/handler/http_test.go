package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gemini-demo/internal/auth"
	"gemini-demo/internal/handler"
	"gemini-demo/internal/models"
	"gemini-demo/internal/util"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTest creates a new in-memory DB, auth service, and template set for testing.
func setupTest(t *testing.T) (*gorm.DB, *auth.AuthService, *template.Template) {
	// Set up environment variable for ProjectRoot
	os.Setenv("GEMINI_TEST_ROOT", "c:\\wk\\testGemini")
	t.Cleanup(func() { os.Unsetenv("GEMINI_TEST_ROOT") })

	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(&models.SiteSetting{}, &models.MenuItemDB{}, &models.Page{})
	assert.NoError(t, err)

	// Initialize Viper for session key
	vp := viper.New()
	vp.Set("auth.session_key", "super-secret-key-for-testing")
	authService := auth.NewAuthServiceWithViper(vp)

	// Parse templates
	projectRoot := util.ProjectRoot("")
	templates, err := parseTemplates(filepath.Join(projectRoot, "templates"))
	assert.NoError(t, err)

	return db, authService, templates
}

// parseTemplates is a helper function to parse templates for tests.
func parseTemplates(templateDir string) (*template.Template, error) {
	var templateFiles []string
	err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			templateFiles = append(templateFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	funcMap := template.FuncMap{
		"getTemplateName": func(r *http.Request) string {
			// This is a simplified version for testing.
			return getTemplateName(r.URL.Path)
		},
	}

	// Parse the files
	templates, err := template.New("").Funcs(funcMap).ParseFiles(templateFiles...)
	if err != nil {
		return nil, err
	}
	return templates, nil
}

func getTemplateName(path string) string {
	if path == "/" {
		return "index.html"
	}
	if strings.HasSuffix(path, "/") {
		path = path + "index.html"
	}
	// This logic is based on how getTemplateName is used in the handler
	// It might need adjustment if your actual implementation is different.
	name := filepath.Base(path)
	if name == "." || name == "/" {
		return "index.html"
	}
	// Ensure it returns the correct template name for admin pages
	if strings.HasPrefix(path, "/admin/") && !strings.HasSuffix(name, ".html") {
		return "admin_" + name + ".html"
	}
	return name
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
	db, _, templates := setupTest(t)
	h := &handler.Handler{DB: db, Templates: templates}

	// Insert test data
	db.Create(&models.SiteSetting{Key: "Title", Value: "Test Title"})
	db.Create(&models.SiteSetting{Key: "Description", Value: "Test Description"})
	db.Create(&models.MenuItemDB{URL: "/home", Text: "Home", Order: 0})

	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.IndexHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "<title>Test Title</title>")
}

func TestPageHandler_Success(t *testing.T) {
	db, _, templates := setupTest(t)
	h := &handler.Handler{DB: db, Templates: templates}

	// Insert test data
	pageName := "test-page"
	db.Create(&models.Page{Name: pageName, Title: "Test Page Title", Message: "Test Page Message"})

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
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Test Page Title")
	assert.Contains(t, rr.Body.String(), "Test Page Message")
}

func TestPageHandler_NotFound(t *testing.T) {
	db, _, templates := setupTest(t)
	h := &handler.Handler{DB: db, Templates: templates}

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
	assert.Contains(t, rr.Body.String(), "Page not found")
}

// func TestPageHandler_InternalError(t *testing.T) {
// 	// Mock DB to return an error
// 	db, _, templates := setupTest(t)
// 	// Simulate a DB error by closing the connection
// 	sqlDB, _ := db.DB()
// 	sqlDB.Close()

// 	h := &handler.Handler{DB: db, Templates: templates}

// 	pageName := "any-page"
// 	req, err := http.NewRequest("GET", fmt.Sprintf("/page/%s", pageName), nil)
// 	assert.NoError(t, err)

// 	vars := map[string]string{
// 		"name": pageName,
// 	}
// 	req = mux.SetURLVars(req, vars)

// 	rr := httptest.NewRecorder()
// 	h.PageHandler(rr, req)

// 	assert.Equal(t, http.StatusInternalServerError, rr.Code)
// 	assert.Contains(t, rr.Body.String(), "Internal Server Error")
// }

func TestAboutHandler(t *testing.T) {
	_, _, templates := setupTest(t)
	h := &handler.Handler{Templates: templates}

	req, err := http.NewRequest("GET", "/about", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	h.AboutHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "This is the about page.")
}

func TestUpdatePageHandler_Success(t *testing.T) {
	db, _, _ := setupTest(t)
	h := &handler.Handler{DB: db}

	// Mock data
	pageName := "home"
	updatedPage := models.Page{
		Name:        pageName,
		Title:       "Updated Home Title",
		Description: "Updated Home Description",
		Message:     "Updated Home Message",
	}
	existingPage := models.Page{
		Name:        pageName,
		Title:       "Original Home Title",
		Description: "Original Home Description",
		Message:     "Original Home Message",
	}

	db.Create(&existingPage)

	// Create request body
	body, err := json.Marshal(updatedPage)
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
	var responsePage models.Page
	err = json.NewDecoder(rr.Body).Decode(&responsePage)
	assert.NoError(t, err)
	assert.Equal(t, updatedPage.Title, responsePage.Title)
}

func TestUpdatePageHandler_NotFound(t *testing.T) {
	db, _, _ := setupTest(t)
	h := &handler.Handler{DB: db}

	pageName := "nonexistent"
	updatedPage := models.Page{Name: pageName, Title: "New Title"}

	// Create request body
	body, err := json.Marshal(updatedPage)
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
	assert.Contains(t, rr.Body.String(), "Page not found")
}

func TestUpdatePageHandler_InvalidBody(t *testing.T) {
	db, _, _ := setupTest(t)
	h := &handler.Handler{DB: db}

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
	assert.Contains(t, rr.Body.String(), "invalid character")
}

func TestUpdatePageHandler_NameMismatch(t *testing.T) {
	db, _, _ := setupTest(t)
	h := &handler.Handler{DB: db}

	pageName := "home"
	updatedPage := models.Page{Name: "mismatch-name", Title: "New Title"}
	body, err := json.Marshal(updatedPage)
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
	assert.Contains(t, rr.Body.String(), "Page name in URL and body do not match")
}

// func TestUpdatePageHandler_InternalError(t *testing.T) {
// 	// Mock DB to return an error during update
// 	db, _, _ := setupTest(t)
// 	// Simulate a DB error by closing the connection
// 	sqlDB, _ := db.DB()
// 	sqlDB.Close()

// 	h := &handler.Handler{DB: db}

// 	pageName := "home"
// 	updatedPage := models.Page{Name: pageName, Title: "New Title"}
// 	body, err := json.Marshal(updatedPage)
// 	assert.NoError(t, err)

// 	req, err := http.NewRequest("PUT", fmt.Sprintf("/pages/%s", pageName), bytes.NewBuffer(body))
// 	assert.NoError(t, err)
// 	req.Header.Set("Content-Type", "application/json")

// 	vars := map[string]string{
// 		"name": pageName,
// 	}
// 	req = mux.SetURLVars(req, vars)

// 	rr := httptest.NewRecorder()
// 	h.UpdatePageHandler(rr, req)

// 	assert.Equal(t, http.StatusInternalServerError, rr.Code)
// 	assert.Contains(t, rr.Body.String(), "Internal Server Error")
// }

func TestLoginHandler_GET_NotLoggedIn(t *testing.T) {
	_, authService, templates := setupTest(t)
	h := &handler.Handler{AuthService: authService, Templates: templates}

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

	_, authService, templates := setupTest(t)
	h := &handler.Handler{AuthService: authService, Templates: templates}

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
	assert.Equal(t, "Login successful", respBody["message"])
}

func TestLoginHandler_POST_InvalidCredentials(t *testing.T) {
	_, authService, templates := setupTest(t)
	h := &handler.Handler{AuthService: authService, Templates: templates}

	credentials := map[string]string{"username": "wronguser", "password": "wrongpass"}
	body, err := json.Marshal(credentials)
	assert.NoError(t, err)

	req, err := http.NewRequest("POST", "/admin/login", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	h.LoginHandler(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid credentials")
}

func TestDashboardHandler_Success(t *testing.T) {
	os.Setenv("ADMIN_USERNAME", "admin")
	os.Setenv("ADMIN_PASSWORD", "password")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	db, authService, templates := setupTest(t)
	h := &handler.Handler{DB: db, AuthService: authService, Templates: templates}

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
	dashboardBody, err := io.ReadAll(dashboardResp.Body)
	assert.NoError(t, err)
	assert.Contains(t, string(dashboardBody), "<h1>Admin Dashboard</h1>")
}

func TestAdminRedirectHandler(t *testing.T) {
	h := &handler.Handler{}

	req, err := http.NewRequest("GET", "/admin", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()

	h.AdminRedirectHandler(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "/admin/dashboard", rr.Header().Get("Location"))
}
