package handler

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"

	"gemini-demo/internal/auth"
	"gemini-demo/internal/models"
	"gemini-demo/internal/i18n" // New import for i18n

	"github.com/gorilla/csrf" // New import for CSRF
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Handler struct {
	DB          *gorm.DB
	AuthService *auth.AuthService
	Templates   *template.Template
	DebugLog    func(format string, v ...interface{})
	Translator  *i18n.Translator // Add Translator field
}

// LoginTemplateData holds data for the login page template.
type LoginTemplateData struct {
	CSRFToken string
	CurrentLang string
}

// DashboardTemplateData holds data for the dashboard page template.
type DashboardTemplateData struct {
	*models.DashboardData // Embed existing data
	CSRFToken             string
	CurrentPath           string // Add CurrentPath field
	CurrentLang           string
}

// AdminPagesTemplateData holds data for the admin pages list template.
type AdminPagesTemplateData struct {
	Pages       []models.Page
	CSRFToken   string
	CurrentPath string
	CurrentLang string
}


// PageCombinedData holds data for a page template, combining page and site data.
type PageCombinedData struct {
	Page *models.Page
	Site *models.Site
	CurrentLang string
}

// IndexTemplateData holds data for the index page template.
type IndexTemplateData struct {
	*models.Site // Embed existing site data
	CurrentLang  string
}

func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	siteData, err := models.GetSiteData(h.DB)
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(h.getLanguage(r), "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting site data: %v", err)
		return
	}

	currentLang := h.getLanguage(r)

	// Dummy Carousel Items for demonstration
	carouselItems := []models.CarouselItem{
		{
			Title:         template.HTML(h.Translator.GetTranslation(currentLang, "carousel_title_1")),
			Description:   template.HTML(h.Translator.GetTranslation(currentLang, "carousel_description_1")),
			ButtonText:    h.Translator.GetTranslation(currentLang, "carousel_button_text_1"),
			ButtonLink:    "#contact",
			BackgroundImage: "/static/images/hero-bg-1.jpg", // Placeholder image
		},
		{
			Title:         template.HTML(h.Translator.GetTranslation(currentLang, "carousel_title_2")),
			Description:   template.HTML(h.Translator.GetTranslation(currentLang, "carousel_description_2")),
			ButtonText:    h.Translator.GetTranslation(currentLang, "carousel_button_text_2"),
			ButtonLink:    "#services",
			BackgroundImage: "/static/images/hero-bg-2.jpg", // Placeholder image
		},
		{
			Title:         template.HTML(h.Translator.GetTranslation(currentLang, "carousel_title_3")),
			Description:   template.HTML(h.Translator.GetTranslation(currentLang, "carousel_description_3")),
			ButtonText:    h.Translator.GetTranslation(currentLang, "carousel_button_text_3"),
			ButtonLink:    "#about",
			BackgroundImage: "/static/images/hero-bg-3.jpg", // Placeholder image
		},
	}

	// Update siteData with carousel items
	siteData.CarouselItems = carouselItems

	data := IndexTemplateData{
		Site:        siteData,
		CurrentLang: currentLang,
	}

	// Execute the template to a buffer first to catch errors before writing to w
	var buf bytes.Buffer
	if err := h.Templates.ExecuteTemplate(&buf, getTemplateName(r), data); err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error executing template (IndexHandler): %v", err)
		return
	}

	// If no error, write the buffer's content to the response writer
	buf.WriteTo(w)
}

func (h *Handler) PageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageName := vars["name"]

	currentLang := h.getLanguage(r)
	// T is now provided by the FuncMap in main.go, no need to pass it in data
	// T := func(key string) string {
	// 	return h.Translator.GetTranslation(currentLang, key)
	// }

	pageData, err := models.GetPageData(h.DB, pageName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, h.Translator.GetTranslation(currentLang, "page_not_found"), http.StatusNotFound) // Translated
		} else {
			http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		}
		return
	}

	siteData, err := models.GetSiteData(h.DB)
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		log.Printf(h.Translator.GetTranslation(currentLang, "error_getting_site_data")+ ": %v", err) // Translated
		return
	}

	combinedData := PageCombinedData{
		Page: pageData,
		Site: siteData,
		CurrentLang: currentLang,
	}

	// Use pre-parsed templates
	if err := h.Templates.ExecuteTemplate(w, getTemplateName(r), combinedData); err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		log.Printf(h.Translator.GetTranslation(currentLang, "error_executing_template")+ ": %v", err) // Translated
		return
	}
}

func (h *Handler) AboutHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)

	siteData, err := models.GetSiteData(h.DB)
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		h.DebugLog("Error getting site data for about page: %v", err)
		return
	}

	data := struct {
		*models.Site // Embed Site data
		CurrentLang string
	}{
		Site: siteData,
		CurrentLang: currentLang,
	}

	if err := h.Templates.ExecuteTemplate(w, "about.html", data); err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		h.DebugLog("Error executing about.html template: %v", err)
	}
}

func (h *Handler) UpdatePageHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)
	// T is now provided by the FuncMap in main.go, no need to pass it in data
	// T := func(key string) string {
	// 	return h.Translator.GetTranslation(currentLang, key)
	// }

	// Log headers for CSRF debugging
	h.DebugLog("UpdatePageHandler: Referer: %s", r.Header.Get("Referer"))
	h.DebugLog("UpdatePageHandler: Origin: %s", r.Header.Get("Origin"))
	h.DebugLog("UpdatePageHandler: X-CSRF-Token header: %s", r.Header.Get("X-CSRF-Token"))

	vars := mux.Vars(r)
	pageName := vars["name"]

	var updatedPage models.Page
	err := json.NewDecoder(r.Body).Decode(&updatedPage)
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "invalid_request_body"), http.StatusBadRequest) // Translated
		return
	}

	if updatedPage.Name != pageName {
		http.Error(w, h.Translator.GetTranslation(currentLang, "page_name_mismatch"), http.StatusBadRequest) // Translated
		return
	}

	existingPage, err := models.GetPageData(h.DB, pageName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, h.Translator.GetTranslation(currentLang, "page_not_found"), http.StatusNotFound) // Translated
		} else {
			http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		}
		return
	}

	existingPage.Title = updatedPage.Title
	existingPage.Description = updatedPage.Description
	existingPage.Message = updatedPage.Message

	if err := existingPage.UpdatePage(h.DB); err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": h.Translator.GetTranslation(currentLang, "login_successful")}) // Translated (using login_successful as a generic success message for now)
}

// LoginHandler handles admin login requests.
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)
	// T is now provided by the FuncMap in main.go, no need to pass it in data
	// T := func(key string) string {
	// 	return h.Translator.GetTranslation(currentLang, key)
	// }

	if h.AuthService.IsLoggedIn(r) {
		http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		token := csrf.Token(r)
		h.DebugLog("LoginHandler(GET): Generated CSRF token: %s", token)
		data := LoginTemplateData{
			CSRFToken: token,
			CurrentLang: currentLang,
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
	h.DebugLog("LoginHandler: X-CSRF-Token header: %s", r.Header.Get("X-CSRF-Token"))
	h.DebugLog("LoginHandler: CSRF token from context (for template): %s", csrf.Token(r))

	err := json.NewDecoder(r.Body).Decode(&credentials)
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "invalid_request_body"), http.StatusBadRequest) // Translated
		return
	}

	log.Printf("Login attempt for user: %s", credentials.Username)
	if h.AuthService.Authenticate(credentials.Username, credentials.Password) {
		err := h.AuthService.Login(w, r)
		if err != nil {
			http.Error(w, h.Translator.GetTranslation(currentLang, "failed_to_login"), http.StatusInternalServerError) // Translated
			return
		}
		log.Printf("Login successful for user: %s", credentials.Username)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": h.Translator.GetTranslation(currentLang, "login_successful")}) // Translated
	} else {
		log.Printf("Login failed for user: %s", credentials.Username)
		http.Error(w, h.Translator.GetTranslation(currentLang, "invalid_credentials"), http.StatusUnauthorized) // Translated
	}
}

// DashboardHandler displays system information after login.
func (h *Handler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)

	data, err := models.GetDashboardData(h.DB)
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		log.Printf(h.Translator.GetTranslation(currentLang, "error_getting_site_data")+ ": %v", err) // Translated
		return
	}

	templateData := DashboardTemplateData{
		DashboardData: data,
		CSRFToken:     csrf.Token(r),
		CurrentPath:   r.URL.Path, // Pass the current request path
		CurrentLang:   currentLang,
	}
	// Execute the template to a buffer first to catch errors before writing to w
	var buf bytes.Buffer
	if err := h.Templates.ExecuteTemplate(&buf, getTemplateName(r), templateData); err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf(h.Translator.GetTranslation(currentLang, "error_executing_template")+ ": %v", err) // Add logging for template execution error
		return
	}

	// If no error, write the buffer's content to the response writer
	buf.WriteTo(w)
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

// getLanguage determines the language based on Accept-Language header or a cookie.
func (h *Handler) getLanguage(r *http.Request) string {
	// Check for language cookie first
	if cookie, err := r.Cookie("lang"); err == nil {
		if h.Translator.IsValidLanguage(cookie.Value) {
			return cookie.Value
		}
	}

	// Fallback to Accept-Language header
	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		// Parse Accept-Language header (e.g., "en-US,en;q=0.9,ja;q=0.8")
		// This is a simplified parsing. A more robust solution would handle q-values.
		parts := strings.Split(acceptLang, ",")
		for _, part := range parts {
			lang := strings.Split(part, ";")[0]
			if h.Translator.IsValidLanguage(lang) {
				return lang
			}
		}
	}

	return h.Translator.DefaultLanguage() // Fallback to default language
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
