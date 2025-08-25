package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"os"

	"gemini-demo/internal/handler"
	"gemini-demo/internal/models"
	"gemini-demo/internal/auth"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	
	"gorm.io/driver/sqlite" // Import sqlite driver
	"gorm.io/gorm"
	"github.com/spf13/viper"
)



func TestIndexHandler(t *testing.T) {
	// Set up environment variable for ProjectRoot
	os.Setenv("GEMINI_TEST_ROOT", "c:\\wk\\testGemini")
	defer os.Unsetenv("GEMINI_TEST_ROOT")

	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{}) // Changed from "file::memory:?cache=shared"
	assert.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(&models.SiteSetting{}, &models.MenuItemDB{})
	assert.NoError(t, err)

	// Insert test data
	db.Create(&models.SiteSetting{Key: "Title", Value: "Test Title"})
	db.Create(&models.SiteSetting{Key: "Description", Value: "Test Description"})
	db.Create(&models.MenuItemDB{URL: "/home", Text: "Home", Order: 0})

	// Create a handler instance with the real DB
	h := &handler.Handler{DB: db, AuthService: nil}

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	h.IndexHandler(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "<title>Test Title</title>") // Check if template rendered correctly
}

func TestAboutHandler(t *testing.T) {
	// Create a handler instance (mocks not strictly needed for this simple handler)
	h := &handler.Handler{}

	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/about", nil)
	assert.NoError(t, err)

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	h.AboutHandler(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "This is the about page.")
}

func TestUpdatePageHandler_Success(t *testing.T) {
	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(&models.Page{})
	assert.NoError(t, err)

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

	// Create a handler instance with the real DB
	h := &handler.Handler{DB: db, AuthService: nil}

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

func TestUpdatePageHandler_PageNotFound(t *testing.T) {
	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(&models.Page{})
	assert.NoError(t, err)

	pageName := "nonexistent"
	updatedPage := models.Page{Name: pageName, Title: "New Title"}

	// Create a handler instance with the real DB
	h := &handler.Handler{DB: db, AuthService: nil}

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

func TestLoginHandler_GET_NotLoggedIn(t *testing.T) {
	// Set up environment variables for AuthService
	os.Setenv("ADMIN_USERNAME", "testuser")
	os.Setenv("ADMIN_PASSWORD", "testpass")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	// Initialize Viper for session key
	vp := viper.New()
	vp.Set("auth.session_key", "super-secret-key")

	// Create a real AuthService instance
	authService := auth.NewAuthServiceWithViper(vp)

	// Create a handler instance with the real AuthService
	h := &handler.Handler{DB: nil, AuthService: authService}

	// Create a request
	req, err := http.NewRequest("GET", "/admin/login", nil)
	assert.NoError(t, err)

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	h.LoginHandler(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code) // Should render login page
	assert.Contains(t, rr.Body.String(), "<title>Admin Login</title>") // Assuming login page has this title
}



func TestLoginHandler_POST_Success(t *testing.T) {
	// Set up environment variables for AuthService
	os.Setenv("ADMIN_USERNAME", "testuser")
	os.Setenv("ADMIN_PASSWORD", "testpass")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	// Initialize Viper for session key
	vp := viper.New()
	vp.Set("auth.session_key", "super-secret-key")

	// Create a real AuthService instance
	authService := auth.NewAuthServiceWithViper(vp)

	// Create a handler instance with the real AuthService
	h := &handler.Handler{DB: nil, AuthService: authService}

	// Create request body
	credentials := map[string]string{"username": "testuser", "password": "testpass"}
	body, err := json.Marshal(credentials)
	assert.NoError(t, err)

	// Create a request
	req, err := http.NewRequest("POST", "/admin/login", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	h.LoginHandler(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Login successful")
}

func TestLoginHandler_POST_InvalidCredentials(t *testing.T) {
	// Set up environment variables for AuthService
	os.Setenv("ADMIN_USERNAME", "testuser")
	os.Setenv("ADMIN_PASSWORD", "testpass")
	defer os.Unsetenv("ADMIN_USERNAME")
	defer os.Unsetenv("ADMIN_PASSWORD")

	// Initialize Viper for session key
	vp := viper.New()
	vp.Set("auth.session_key", "super-secret-key")

	// Create a real AuthService instance
	authService := auth.NewAuthServiceWithViper(vp)

	// Create a handler instance with the real AuthService
	h := &handler.Handler{DB: nil, AuthService: authService}

	// Create request body
	credentials := map[string]string{"username": "wronguser", "password": "wrongpass"}
	body, err := json.Marshal(credentials)
	assert.NoError(t, err)

	// Create a request
	req, err := http.NewRequest("POST", "/admin/login", bytes.NewBuffer(body))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	h.LoginHandler(rr, req)

	// Assertions
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid credentials")
}

func TestDashboardHandler_Success(t *testing.T) {
	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(&models.SiteSetting{}, &models.MenuItemDB{}, &models.Page{})
	assert.NoError(t, err)

	// Clear existing site settings to ensure a clean state
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.SiteSetting{})

	// Insert test data
	db.Create(&models.SiteSetting{Key: "Title", Value: "Dashboard Title"})
	db.Create(&models.SiteSetting{Key: "Description", Value: "Dashboard Description"})
	db.Create(&models.Page{}) // Create a page to make page count > 0

	// Create a handler instance with the real DB
	h := &handler.Handler{DB: db, AuthService: nil}

	// Create a request
	req, err := http.NewRequest("GET", "/admin/dashboard", nil)
	assert.NoError(t, err)

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	h.DashboardHandler(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Dashboard Title")
	assert.Contains(t, rr.Body.String(), "Total Pages: 1") // Assuming one page was created
}

func TestAdminRedirectHandler(t *testing.T) {
	// Create a handler instance (mocks not strictly needed for this simple handler)
	h := &handler.Handler{}

	// Create a request
	req, err := http.NewRequest("GET", "/admin", nil)
	assert.NoError(t, err)

	// Create a ResponseRecorder
	rr := httptest.NewRecorder()

	// Serve the HTTP request
	h.AdminRedirectHandler(rr, req)

	// Assertions
	assert.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "/admin/dashboard", rr.Header().Get("Location"))
}