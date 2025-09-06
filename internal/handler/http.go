package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"gemini-demo/internal/auth"
	"gemini-demo/internal/i18n"
	"gemini-demo/internal/models"
	"gemini-demo/internal/util"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Handler struct {
	Store       models.DataStore
	AuthService *auth.AuthService
	Templates   map[string]*template.Template // Changed to map
	DebugLog    func(format string, v ...interface{})
	Translator  *i18n.Translator
	DebugMode   bool
}

// ContentTemplater defines an interface for data structures that can specify a content template name.
type ContentTemplater interface {
	GetContentTemplateName() string
}



// GetContentTemplateName implements ContentTemplater for PagesListTemplateData.
func (d PagesListTemplateData) GetContentTemplateName() string {
	return d.ContentTemplateName
}



func (h *Handler) renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data interface{}) {
	var currentTemplates map[string]*template.Template
	var err error

	if h.DebugMode {
		currentTemplates, err = util.ParseTemplates(h.Translator)
		if err != nil {
			log.Printf("Error parsing templates in debug mode: %v", err)
			http.Error(w, h.Translator.GetTranslation(h.getLanguage(r), "error_processing_template"), http.StatusInternalServerError)
			return
		}
	} else {
		currentTemplates = h.Templates
	}

	// Debugging: Print keys in currentTemplates
	log.Printf("Available templates in map:")
	for k := range currentTemplates {
		log.Printf("- %s", k)
	}

	tmpl, ok := currentTemplates[templateName]
	if !ok {
		http.Error(w, h.Translator.GetTranslation(h.getLanguage(r), "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Template %s not found in map", templateName)
		return
	}

	var executeTemplateName string
	if ct, ok := data.(ContentTemplater); ok {
		executeTemplateName = ct.GetContentTemplateName()
	}
	if executeTemplateName == "" {
		if strings.HasPrefix(templateName, "admin/") {
			executeTemplateName = templateName
		} else {
			executeTemplateName = "base" // Fallback to "base" for non-admin
		}
	}

	var buf bytes.Buffer
	// Execute the specified content template name (e.g., "index_content", "pages_content", "page_content")
	// or fallback to "base" if not specified.
	if err := tmpl.ExecuteTemplate(&buf, executeTemplateName, data); err != nil {
		http.Error(w, h.Translator.GetTranslation(h.getLanguage(r), "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error executing template (%s) with content template (%s): %v", templateName, executeTemplateName, err)
		return
	}
	buf.WriteTo(w)
}

// LoginTemplateData holds data for the login page template.
type LoginTemplateData struct {
	CSRFToken   string
	CurrentLang string
}

// DashboardTemplateData holds data for the dashboard page template.
type DashboardTemplateData struct {
	*models.DashboardData // Embed existing data
	CSRFToken             string
	CurrentPath           string // Add CurrentPath field
	CurrentLang           string
}



// AdminEditPageTemplateData holds data for the admin edit page template.
type AdminEditPageTemplateData struct {
	Page        *models.Page
	CSRFToken   string
	CurrentPath string
	CurrentLang string
	Title       string
	Message     template.HTML
	IsNew       bool // Flag for new page creation
}



// AdminPagesTemplateData holds data for the admin pages list template.
type AdminPagesTemplateData struct {
	Pages       []models.Page
	CSRFToken   string
	CurrentPath string
	CurrentLang string
	Title       string
}



// PageCombinedData holds data for a page template, combining page and site data.
type PageCombinedData struct {
	Page                *models.Page
	Site                *models.Site
	CurrentLang         string
}

// IndexTemplateData holds data for the index page template.
type IndexTemplateData struct {
	Site                *models.Site // Explicit field
	CurrentLang         string
}

// PagesListTemplateData holds data for the pages list page template.
type PagesListTemplateData struct {
	Pages               []models.Page
	Site                *models.Site
	CurrentLang         string
	ContentTemplateName string
}

func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	siteData, err := h.Store.GetSiteData()
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(h.getLanguage(r), "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting site data: %v", err)
		return
	}

	currentLang := h.getLanguage(r)
	h.DebugLog("IndexHandler: currentLang = %s", currentLang) // Add this line

	// Dummy Carousel Items for demonstration
	carouselItems := []models.CarouselItem{
		{
			Title:         template.HTML(h.Translator.GetTranslation(currentLang, "carousel_title_1")),
			Description:   template.HTML(h.Translator.GetTranslation(currentLang, "carousel_description_1")),
			ButtonText:    h.Translator.GetTranslation(currentLang, "carousel_button_text_1"),
			ButtonLink:    "#contact",
			BackgroundImage: "/static/images/hero-bg-1.jpg", // Placeholder image
			Active:        true,
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
		Site:                siteData,
		CurrentLang:         currentLang,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.renderTemplate(w, r, getTemplateName(r), data)
}

func (h *Handler) PagesListHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)

	pages, err := h.Store.GetAllPages()
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting all pages: %v", err)
		return
	}

	siteData, err := h.Store.GetSiteData()
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting site data: %v", err)
		return
	}

	data := PagesListTemplateData{
		Pages:               pages,
		Site:                siteData,
		CurrentLang:         currentLang,
	}

	h.renderTemplate(w, r, "pages.html", data)
}

func (h *Handler) PageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageName := vars["name"]

	currentLang := h.getLanguage(r)

	pageData, err := h.Store.GetPageData(pageName)
	if err != nil {
		if err.Error() == "record not found" {
			http.Error(w, h.Translator.GetTranslation(currentLang, "page_not_found"), http.StatusNotFound) // Translated
		} else {
			http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		}
		return
	}

	siteData, err := h.Store.GetSiteData()
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		log.Printf(h.Translator.GetTranslation(currentLang, "error_getting_site_data")+ ": %v", err) // Translated
		return
	}

	combinedData := PageCombinedData{
		Page:                pageData,
		Site:                siteData,
		CurrentLang:         currentLang,
	}

	h.renderTemplate(w, r, getTemplateName(r), combinedData)
}

func (h *Handler) AboutHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)
	h.DebugLog("AboutHandler: currentLang = %s", currentLang) // Add this line

	siteData, err := h.Store.GetSiteData()
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		h.DebugLog("Error getting site data for about page: %v", err)
		return
	}

	data := struct {
		Site *models.Site // Explicit field
		CurrentLang string
	}{
		Site: siteData,
		CurrentLang: currentLang,
	}

	h.renderTemplate(w, r, "about.html", data)
}




func (h *Handler) UpdatePageHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)

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

	existingPage, err := h.Store.GetPageData(pageName)
	if err != nil {
		if err.Error() == "record not found" {
			http.Error(w, h.Translator.GetTranslation(currentLang, "page_not_found"), http.StatusNotFound) // Translated
		} else {
			http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		}
		return
	}

	existingPage.Title = updatedPage.Title
	existingPage.Description = updatedPage.Description
	existingPage.Message = updatedPage.Message

	if err := h.Store.UpdatePage(existingPage); err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(existingPage)
}

// LoginHandler handles admin login requests.
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)

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
		h.renderTemplate(w, r, getTemplateName(r), data)
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

	data, err := h.Store.GetDashboardData()
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		log.Printf(h.Translator.GetTranslation(currentLang, "error_getting_site_data")+ ": %v", err) // Translated
		return
	}

	templateData := DashboardTemplateData{
		DashboardData:       data,
		CSRFToken:           csrf.Token(r),
		CurrentPath:         r.URL.Path, // Pass the current request path
		CurrentLang:         currentLang,
	}
	h.renderTemplate(w, r, getTemplateName(r), templateData)
}

// PagesHandler displays the list of pages in the admin panel.
func (h *Handler) PagesHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)
	pages, err := h.Store.GetAllPages()
	if err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting all pages: %v", err)
		return
	}

	data := AdminPagesTemplateData{
		Title:       h.Translator.GetTranslation(currentLang, "pages_menu"),
		Pages:       pages,
		CSRFToken:   csrf.Token(r),
		CurrentPath: r.URL.Path,
		CurrentLang: currentLang,
	}

	h.renderTemplate(w, r, getTemplateName(r), data)
}

// AdminEditPageHandler handles the display of the admin page edit form.
func (h *Handler) AdminEditPageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageName := vars["name"]
	currentLang := h.getLanguage(r)

	page, err := h.Store.GetPageData(pageName)
	if err != nil {
		if err.Error() == "record not found" {
			http.NotFound(w, r)
		} else {
			http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		}
		return
	}

	data := AdminEditPageTemplateData{
		Title:       h.Translator.GetTranslation(currentLang, "edit_page_title"),
		Page:        page,
		CSRFToken:   csrf.Token(r),
		CurrentPath: r.URL.Path,
		CurrentLang: currentLang,
		Message:     template.HTML(page.Message),
	}

	h.renderTemplate(w, r, "admin/admin_edit.html", data)
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
		http.Error(w, h.Translator.GetTranslation(h.getLanguage(r), "failed_to_log_out"), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/login", http.StatusFound) // Redirect to login page
}

// ImageUploadHandler handles image uploads for the editor.
func (h *Handler) ImageUploadHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB
		http.Error(w, h.Translator.GetTranslation(h.getLanguage(r), "unable_to_parse_form"), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("image")
	if err != nil {
					http.Error(w, h.Translator.GetTranslation(h.getLanguage(r), "unable_to_get_image_from_form"), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate a random filename
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
					http.Error(w, h.Translator.GetTranslation(h.getLanguage(r), "failed_to_generate_random_filename"), http.StatusInternalServerError)
		return
	}
	filename := fmt.Sprintf("%x%s", randomBytes, filepath.Ext(handler.Filename))

	// Create the file
	dst, err := os.Create(filepath.Join("static", "images", filename))
	if err != nil {
					http.Error(w, h.Translator.GetTranslation(h.getLanguage(r), "unable_to_create_file_for_writing"), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination file
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, h.Translator.GetTranslation(h.getLanguage(r), "unable_to_save_file"), http.StatusInternalServerError)
		return
	}

	// Return the URL of the uploaded file
	json.NewEncoder(w).Encode(map[string]string{
		"url": "/static/images/" + filename,
	})
}

// AdminNewPageHandler handles both displaying the form and creating a new page.
func (h *Handler) AdminNewPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.createPagePost(w, r)
		return
	}
	h.showNewPageForm(w, r)
}

func (h *Handler) showNewPageForm(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)
	data := AdminEditPageTemplateData{
		Title:       h.Translator.GetTranslation(currentLang, "new_page_title"),
		Page:        &models.Page{},
		CSRFToken:   csrf.Token(r),
		CurrentPath: r.URL.Path,
		CurrentLang: currentLang,
		Message:     "",
		IsNew:       true,
	}

	h.renderTemplate(w, r, "admin/admin_edit.html", data)
}

func (h *Handler) createPagePost(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)
	var newPage models.Page
	if err := json.NewDecoder(r.Body).Decode(&newPage); err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "invalid_request_body"), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(newPage.Name) == "" || strings.TrimSpace(newPage.Title) == "" {
		http.Error(w, h.Translator.GetTranslation(currentLang, "page_name_title_required"), http.StatusBadRequest)
		return
	}

	_, err := h.Store.GetPageData(newPage.Name)
	if err == nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "page_exists"), http.StatusConflict)
		return
	}

	if err := h.Store.CreatePage(&newPage); err != nil {
		http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error creating page: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newPage)
}

// DeletePageHandler handles the deletion of a page.
func (h *Handler) DeletePageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageName := vars["name"]
	currentLang := h.getLanguage(r)

	err := h.Store.DeletePage(pageName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, h.Translator.GetTranslation(currentLang, "page_not_found"), http.StatusNotFound)
		} else {
			http.Error(w, h.Translator.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		}
		log.Printf("Error deleting page %s: %v", pageName, err)
		return
	}

	w.WriteHeader(http.StatusOK)
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
	h.DebugLog("Accept-Language header: %s", acceptLang)
	if acceptLang != "" {
		// Parse Accept-Language header (e.g., "en-US,en;q=0.9,ja;q=0.8")
		// This is a simplified parsing. A more robust solution would handle q-values.
		parts := strings.Split(acceptLang, ",")
		for _, part := range parts {
			lang := strings.Split(part, ";")[0]
			h.DebugLog("Parsed language: %s, IsValid: %t", lang, h.Translator.IsValidLanguage(lang))
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
	if strings.HasPrefix(path, "/admin/") {
		// Handle admin paths specifically
		// e.g., /admin/dashboard -> admin/admin_dashboard.html
		// e.g., /admin/pages -> admin/admin_pages.html
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(parts) == 2 {
			return fmt.Sprintf("%s/%s_%s.html", parts[0], parts[0], parts[1])
		}
		return strings.TrimPrefix(path, "/") + ".html" // Fallback for other admin paths
	}
	// For other top-level paths like /pages, /about
	// Example: "/pages" -> "pages.html"
	// Example: "/about" -> "about.html"
	return strings.TrimPrefix(path, "/") + ".html"
}