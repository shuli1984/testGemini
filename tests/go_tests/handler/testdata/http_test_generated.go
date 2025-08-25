package handler_test

import (
	"bytes"
	"encoding/json"
	"gemini-demo/internal/database"
	"gemini-demo/internal/handler"
	"gemini-demo/internal/models"
	"gemini-demo/tests/testutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	testutil.SetupViper()
	db, sqlDB, err := database.InitDB()
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}
	models.AutoMigrateAndSeed(db)
	return db, func() {
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove("./gemini.db")
	}
}

func TestUpdatePageHandler(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	h := &handler.Handler{DB: db}
	r := mux.NewRouter()
	r.Put("/pages/{name}", h.UpdatePageHandler)

	// Test Case 1: Successful Update
	t.Run("Successful Update", func(t *testing.T) {
		pageName := "home"
		updatedPage := models.Page{
			Name:        pageName,
			Title:       "Updated Home Title",
			Description: "Updated Home Description",
			Message:     "Updated Home Message",
		}
		body, _ := json.Marshal(updatedPage)
		req := httptest.NewRequest("PUT", "/pages/"+pageName, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var responsePage models.Page
		json.NewDecoder(rr.Body).Decode(&responsePage)
		if responsePage.Title != updatedPage.Title || responsePage.Description != updatedPage.Description || responsePage.Message != updatedPage.Message {
			t.Errorf("handler returned unexpected body: got %+v want %+v", responsePage, updatedPage)
		}
	})

	// Test Case 2: Mismatched Page Names
	t.Run("Mismatched Page Names", func(t *testing.T) {
		pageName := "home"
		updatedPage := models.Page{
			Name:        "about", // Mismatched name
			Title:       "Updated Home Title",
			Description: "Updated Home Description",
			Message:     "Updated Home Message",
		}
		body, _ := json.Marshal(updatedPage)
		req := httptest.NewRequest("PUT", "/pages/"+pageName, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	// Test Case 3: Non-existent Page
	t.Run("Non-existent Page", func(t *testing.T) {
		pageName := "nonexistent"
		updatedPage := models.Page{
			Name:        pageName,
			Title:       "Nonexistent Title",
			Description: "Nonexistent Description",
			Message:     "Nonexistent Message",
		}
		body, _ := json.Marshal(updatedPage)
		req := httptest.NewRequest("PUT", "/pages/"+pageName, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	// Test Case 4: Invalid JSON Body
	t.Run("Invalid JSON Body", func(t *testing.T) {
		pageName := "home"
		req := httptest.NewRequest("PUT", "/pages/"+pageName, strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	// Test Case 5: Database Error during update (simulate by closing DB)
	t.Run("Database Error", func(t *testing.T) {
		pageName := "home"
		updatedPage := models.Page{
			Name:        pageName,
			Title:       "Updated Home Title",
			Description: "Updated Home Description",
			Message:     "Updated Home Message",
		}
		body, _ := json.Marshal(updatedPage)
		req := httptest.NewRequest("PUT", "/pages/"+pageName, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Close the database to simulate an error
		sqlDB, _ := db.DB()
		sqlDB.Close()

		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})
}

func TestLoginHandler(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	h := &handler.Handler{DB: db}
	r := mux.NewRouter()
	r.Post("/admin/login", h.LoginHandler)

	// Test Case 1: Successful Login
	t.Run("Successful Login", func(t *testing.T) {
		os.Setenv("ADMIN_USERNAME", "testadmin")
		os.Setenv("ADMIN_PASSWORD", "testpassword")
		defer os.Unsetenv("ADMIN_USERNAME")
		defer os.Unsetenv("ADMIN_PASSWORD")

		credentials := map[string]string{"username": "testadmin", "password": "testpassword"}
		body, _ := json.Marshal(credentials)
		req := httptest.NewRequest("POST", "/admin/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if rr.Header().Get("Set-Cookie") == "" {
			t.Error("Expected Set-Cookie header, but got none")
		}
	})

	// Test Case 2: Failed Login (Incorrect Credentials)
	t.Run("Failed Login - Incorrect Credentials", func(t *testing.T) {
		os.Setenv("ADMIN_USERNAME", "testadmin")
		os.Setenv("ADMIN_PASSWORD", "testpassword")
		defer os.Unsetenv("ADMIN_USERNAME")
		defer os.Unsetenv("ADMIN_PASSWORD")

		credentials := map[string]string{"username": "wronguser", "password": "wrongpass"}
		body, _ := json.Marshal(credentials)
		req := httptest.NewRequest("POST", "/admin/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	// Test Case 3: Invalid Request Body
	t.Run("Invalid Request Body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/admin/login", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

func TestDashboardHandler(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	h := &handler.Handler{DB: db}
	r := mux.NewRouter()
	r.Get("/admin/dashboard", h.DashboardHandler)

	// Test Case 1: Successful Dashboard Display
	t.Run("Successful Dashboard Display", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin/dashboard", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		// Further checks can be added here to verify content of the dashboard page
	})

	// Test Case 2: Database Error during page count (simulate by closing DB)
	t.Run("Database Error during Page Count", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin/dashboard", nil)
		rr := httptest.NewRecorder()

		// Close the database to simulate an error
		sqlDB, _ := db.DB()
		sqlDB.Close()

		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})
}

func TestAuthMiddleware(t *testing.T) {
	// A simple handler that the middleware will wrap
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authorized"))
	})

	// Test Case 1: Valid Session Cookie
	t.Run("Valid Session Cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: "loggedIn"})
		rr := httptest.NewRecorder()

		handler.AuthMiddleware(nextHandler).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if rr.Body.String() != "Authorized" {
			t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), "Authorized")
		}
	})

	// Test Case 2: No Session Cookie (Redirection)
	t.Run("No Session Cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		rr := httptest.NewRecorder()

		handler.AuthMiddleware(nextHandler).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
		}
		if location := rr.Header().Get("Location"); location != "/admin/login" {
			t.Errorf("handler returned wrong redirect location: got %v want %v", location, "/admin/login")
		}
	})

	// Test Case 3: Invalid Session Cookie (Redirection)
	t.Run("Invalid Session Cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "session_token", Value: "invalid"})
		rr := httptest.NewRecorder()

		handler.AuthMiddleware(nextHandler).ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
		}
		if location := rr.Header().Get("Location"); location != "/admin/login" {
			t.Errorf("handler returned wrong redirect location: got %v want %v", location, "/admin/login")
		}
	})
}

func TestAdminRedirectHandler(t *testing.T) {
	db, teardown := setupTestDB(t)
	defer teardown()

	h := &handler.Handler{DB: db}
	r := mux.NewRouter()
	r.Get("/admin", h.AdminRedirectHandler)

	// Test Case 1: Successful Redirection
	t.Run("Successful Redirection", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
		}
		if location := rr.Header().Get("Location"); location != "/admin/dashboard" {
			t.Errorf("handler returned wrong redirect location: got %v want %v", location, "/admin/dashboard")
		}
	})
}
