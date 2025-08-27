package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"gemini-demo/internal/auth"
	"gemini-demo/internal/models"

	"github.com/gorilla/csrf" // New import for CSRF
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Handler struct {
	DB          *gorm.DB
	AuthService *auth.AuthService
	Templates   *template.Template
}

// LoginTemplateData holds data for the login page template.
type LoginTemplateData struct {
	CSRFToken string
}

// DashboardTemplateData holds data for the dashboard page template.
type DashboardTemplateData struct {
	*models.DashboardData // Embed existing data
	CSRFToken             string
}

// PageCombinedData holds data for a page template, combining page and site data.
type PageCombinedData struct {
	Page *models.Page
	Site *models.Site
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

	siteData, err := models.GetSiteData(h.DB)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error getting site data for page: %v", err)
		return
	}

	combinedData := PageCombinedData{
		Page: pageData,
		Site: siteData,
	}

	// Use pre-parsed templates
	if err := h.Templates.ExecuteTemplate(w, getTemplateName(r), combinedData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error executing template (PageHandler): %v", err)
		return
	}
}

func (h *Handler) AboutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "This is the about page.")
}

func (h *Handler) UpdatePageHandler(w http.ResponseWriter, r *http.Request) {
	// Log headers for CSRF debugging
	log.Printf("UpdatePageHandler: Referer: %s", r.Header.Get("Referer"))
	log.Printf("UpdatePageHandler: Origin: %s", r.Header.Get("Origin"))
	log.Printf("UpdatePageHandler: X-CSRF-Token header: %s", r.Header.Get("X-CSRF-Token"))

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
		token := csrf.Token(r)
		log.Printf("LoginHandler(GET): Generated CSRF token: %s", token)
		data := LoginTemplateData{
			CSRFToken: token,
		}
		if err := h.Templates.ExecuteTemplate(w, getTemplateName(r), data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	// Log CSRF tokens for debugging
	log.Printf("LoginHandler: X-CSRF-Token header: %s", r.Header.Get("X-CSRF-Token"))
	log.Printf("LoginHandler: CSRF token from context (for template): %s", csrf.Token(r))

	err := json.NewDecoder(r.Body).Decode(&credentials)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Login attempt for user: %s", credentials.Username)
	if h.AuthService.Authenticate(credentials.Username, credentials.Password) {
		err := h.AuthService.Login(w, r)
		if err != nil {
			http.Error(w, "Failed to log in", http.StatusInternalServerError)
			return
		}
		log.Printf("Login successful for user: %s", credentials.Username)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
	} else {
		log.Printf("Login failed for user: %s", credentials.Username)
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

	templateData := DashboardTemplateData{
		DashboardData: data,
		CSRFToken:     csrf.Token(r),
	}
	if err := h.Templates.ExecuteTemplate(w, getTemplateName(r), templateData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// AdminRedirectHandler redirects /admin to /admin/dashboard
func (h *Handler) AdminRedirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
}

// LogoutHandler handles admin logout requests.
func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	err := h.AuthService.Logout(w, r)
	if err != nil {
		log.Printf("Error logging out: %v", err)
		http.Error(w, "Failed to log out", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/login", http.StatusFound) // Redirect to login page
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
