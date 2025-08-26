package handler

import (
	"encoding/json"
	"fmt"
	"html/template" // Keep this import for *template.Template type
	"log"
	"net/http"
	"strings" // Keep this import for getTemplateName

	"gemini-demo/internal/auth"
	"gemini-demo/internal/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Handler struct {
	DB          *gorm.DB
	AuthService *auth.AuthService
	Templates   *template.Template // New field for pre-parsed templates
}

func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	siteData, err := models.GetSiteData(h.DB)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error getting site data: %v", err)
		return
	}

	// Use pre-parsed templates
	if err := h.Templates.ExecuteTemplate(w, getTemplateName(r), siteData); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error executing template (IndexHandler): %v", err)
		return
	}
}

func (h *Handler) PageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageName := vars["name"]

	pageData, err := models.GetPageData(h.DB, pageName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Page not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Use pre-parsed templates
	if err := h.Templates.ExecuteTemplate(w, getTemplateName(r), pageData); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error executing template (PageHandler): %v", err)
		return
	}
}

func (h *Handler) AboutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "This is the about page.")
}

func (h *Handler) UpdatePageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageName := vars["name"]

	var updatedPage models.Page
	err := json.NewDecoder(r.Body).Decode(&updatedPage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if updatedPage.Name != pageName {
		http.Error(w, "Page name in URL and body do not match", http.StatusBadRequest)
		return
	}

	existingPage, err := models.GetPageData(h.DB, pageName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Page not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	existingPage.Title = updatedPage.Title
	existingPage.Description = updatedPage.Description
	existingPage.Message = updatedPage.Message

	if err := existingPage.UpdatePage(h.DB); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(existingPage)
}

// LoginHandler handles admin login requests.
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if h.AuthService.IsLoggedIn(r) {
		http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		// Use pre-parsed templates
		if err := h.Templates.ExecuteTemplate(w, getTemplateName(r), nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&credentials)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if h.AuthService.Authenticate(credentials.Username, credentials.Password) {
		err := h.AuthService.Login(w, r)
		if err != nil {
			http.Error(w, "Failed to log in", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
	} else {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
	}
}

// DashboardHandler displays system information after login.
func (h *Handler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	data, err := models.GetDashboardData(h.DB)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error getting dashboard data: %v", err)
		return
	}

	// Use pre-parsed templates
	if err := h.Templates.ExecuteTemplate(w, getTemplateName(r), data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// AdminRedirectHandler redirects /admin to /admin/dashboard
func (h *Handler) AdminRedirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
}

func getTemplateName(r *http.Request) string {
	path := r.URL.Path
	if path == "/" {
		return "index.html"
	}
	if strings.HasPrefix(path, "/page/") {
		return "page.html"
	}
	return strings.Replace(strings.TrimPrefix(path, "/"), "/", "_", 1) + ".html"
}