package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"gemini-demo/internal/auth"
	"gemini-demo/internal/i18n"
	"gemini-demo/internal/models"
	"gemini-demo/internal/translator"
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
	Store          models.DataStore
	AuthService    *auth.AuthService
	Templates      map[string]*template.Template // Changed to map
	DebugLog       func(format string, v ...interface{})
	I18n           *i18n.Translator
	API_Translator translator.Translator
	DebugMode      bool
}

// ContentTemplater defines an interface for data structures that can specify a content template name.
type ContentTemplater interface {
	GetContentTemplateName() string
}

// GetContentTemplateName implements ContentTemplater for PagesListTemplateData.
// func (d PagesListTemplateData) GetContentTemplateName() string {
// 	return d.ContentTemplateName
// }

func (h *Handler) renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data interface{}) {
	var currentTemplates map[string]*template.Template
	var err error

	if h.DebugMode {
		currentTemplates, err = util.ParseTemplates(h.I18n)
		if err != nil {
			log.Printf("Error parsing templates in debug mode: %v", err)
			http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "error_processing_template"), http.StatusInternalServerError)
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
		http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "internal_server_error"), http.StatusInternalServerError)
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
		http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "internal_server_error"), http.StatusInternalServerError)
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
	Page                        *models.Page
	CSRFToken                   string
	CurrentPath                 string
	CurrentLang                 string
	Message                     template.HTML
	IsNew                       bool // Flag for new page creation
	SupportedLanguages          []string
	EditLang                    string
	OriginalContentLanguageCode string // Add this field
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
	Site          *models.Site // Explicit field
	CoreSolutions []models.Page
	CurrentLang   string
	Page          *models.Page // Add Page field for header compatibility
}

// PagesListTemplateData holds data for the pages list page template.
// PagesListTemplateData holds data for the pages list page template.
type PagesListTemplateData struct {
	Page                *models.Page // Add Page field for header compatibility
	Pages               []models.Page
	Site                *models.Site
	CurrentLang         string
	ContentTemplateName string
}

func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	siteData, err := h.Store.GetSiteData()
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting site data: %v", err)
		return
	}

	currentLang := h.getLanguage(r)
	h.DebugLog("IndexHandler: currentLang = %s", currentLang) // Add this line

	// Dummy Carousel Items for demonstration
	carouselItems := []models.CarouselItem{
		{
			Title:         template.HTML(h.I18n.GetTranslation(currentLang, "common.carousel_title_1")),
			Description:   template.HTML(h.I18n.GetTranslation(currentLang, "common.carousel_description_1")),
			ButtonText:    h.I18n.GetTranslation(currentLang, "common.carousel_button_text_1"),
			ButtonLink:    "#contact",
			BackgroundImage: "/static/images/hero-bg-1.jpg", // Placeholder image
			Active:        true,
		},
		{
			Title:         template.HTML(h.I18n.GetTranslation(currentLang, "common.carousel_title_2")),
			Description:   template.HTML(h.I18n.GetTranslation(currentLang, "common.carousel_description_2")),
			ButtonText:    h.I18n.GetTranslation(currentLang, "common.carousel_button_text_2"),
			ButtonLink:    "#services",
			BackgroundImage: "/static/images/hero-bg-2.jpg", // Placeholder image
		},
		{
			Title:         template.HTML(h.I18n.GetTranslation(currentLang, "common.carousel_title_3")),
			Description:   template.HTML(h.I18n.GetTranslation(currentLang, "common.carousel_description_3")),
			ButtonText:    h.I18n.GetTranslation(currentLang, "common.carousel_button_text_3"),
			ButtonLink:    "#about",
			BackgroundImage: "/static/images/hero-bg-3.jpg", // Placeholder image
		},
	}

	// Update siteData with carousel items
	siteData.CarouselItems = carouselItems

	coreSolutions, err := h.Store.GetCoreSolutions(currentLang, h.I18n.DefaultLanguage())
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting core solutions: %v", err)
		return
	}

	data := IndexTemplateData{
		Site:          siteData,
		CoreSolutions: coreSolutions,
		CurrentLang:   currentLang,
		Page:          nil, // Initialize Page to nil for index page
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.renderTemplate(w, r, getTemplateName(r), data)
}

func (h *Handler) PagesListHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)

	pages, err := h.Store.GetAllPages(currentLang, h.I18n.DefaultLanguage())
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting all pages: %v", err)
		return
	}

	siteData, err := h.Store.GetSiteData()
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
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

	pageData, err := h.Store.GetPageData(pageName, currentLang, h.I18n.DefaultLanguage())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, h.I18n.GetTranslation(currentLang, "page_not_found"), http.StatusNotFound) // Translated
		} else {
			http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		}
		return
	}

	siteData, err := h.Store.GetSiteData()
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		log.Printf(h.I18n.GetTranslation(currentLang, "error_getting_site_data")+ ": %v", err) // Translated
		return
	}

	combinedData := PageCombinedData{
		Page:        pageData,
		Site:        siteData,
		CurrentLang: currentLang,
	}

	h.renderTemplate(w, r, getTemplateName(r), combinedData)
}

func (h *Handler) AboutHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)
	h.DebugLog("AboutHandler: currentLang = %s", currentLang) // Add this line

	siteData, err := h.Store.GetSiteData()
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		h.DebugLog("Error getting site data for about page: %v", err)
		return
	}

	data := struct {
		Site *models.Site // Explicit field
		CurrentLang string
		Page *models.Page // Add Page field for header compatibility
	}{
		Site: siteData,
		CurrentLang: currentLang,
		Page: nil, // Initialize Page to nil for about page
	}

	h.renderTemplate(w, r, "about.html", data)
}

// PageUpdatePayload mirrors the structure of the JSON payload sent from the frontend for page updates.
type PageUpdatePayload struct {
	Name           string        `json:"Name"`
	Title          string        `json:"Title"`
	Description    string        `json:"Description"`
	Message        template.HTML `json:"Message"`
	IsCoreSolution bool          `json:"IsCoreSolution"`
	Icon           string        `json:"Icon"`
	LanguageCode   string        `json:"LanguageCode"` // Add LanguageCode to the payload
}

func (h *Handler) UpdatePageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageName := vars["name"]
	currentLang := h.getLanguage(r) // Language of the UI
	log.Printf("UpdatePageHandler started for page: %s, language: %s", pageName, currentLang)

	var payload PageUpdatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("UpdatePageHandler: Error decoding payload for page %s: %v", pageName, err)
		http.Error(w, h.I18n.GetTranslation(currentLang, "invalid_request_body"), http.StatusBadRequest)
		return
	}

	// Determine the language being edited. Prioritize payload's LanguageCode, then query param, then currentLang.
	editLang := payload.LanguageCode
	if editLang == "" {
		editLang = r.URL.Query().Get("lang")
		if editLang == "" {
			editLang = currentLang
		}
	}

	// Basic validation for translation fields
	if strings.TrimSpace(payload.Title) == "" {
		http.Error(w, h.I18n.GetTranslation(currentLang, "title_required"), http.StatusBadRequest)
		return
	}

	// 1. Update the main Page fields (IsCoreSolution, Icon)
	// First, get the existing page to update its non-translation fields.
	existingPage, err := h.Store.GetPageData(pageName, editLang, h.I18n.DefaultLanguage())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, h.I18n.GetTranslation(currentLang, "page_not_found"), http.StatusNotFound)
		} else {
			http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		}
		return
	}

	// Update only the fields that belong to the Page struct
	existingPage.IsCoreSolution = payload.IsCoreSolution
	existingPage.Icon = payload.Icon

	// Save the updated Page (excluding its Content field which is not persisted directly)
	if err := h.Store.UpdatePage(existingPage); err != nil { // Assuming an UpdatePage method exists or will be created
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error updating page: %v", err)
		return
	}

	// 2. Update or create the PageTranslation
	translation := models.PageTranslation{
		Title:        payload.Title,
		Description:  payload.Description,
		Keywords:     "", // Keywords are not in the current form, so leave empty or fetch from existing if needed
		Message:      payload.Message,
		LanguageCode: editLang,
	}

	if err := h.Store.UpdatePageTranslation(existingPage.ID, &translation); err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error updating page translation: %v", err)
		return
	}

	log.Printf("UpdatePageHandler successfully updated page: %s, language: %s", pageName, editLang)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": h.I18n.GetTranslation(currentLang, "save_successful")})
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": h.I18n.GetTranslation(currentLang, "invalid_request_body")})
		return
	}

	log.Printf("Login attempt for user: %s", credentials.Username)
	if h.AuthService.Authenticate(credentials.Username, credentials.Password) {
		err := h.AuthService.Login(w, r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": h.I18n.GetTranslation(currentLang, "failed_to_login")})
			return
		}
		log.Printf("Login successful for user: %s", credentials.Username)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": h.I18n.GetTranslation(currentLang, "login_successful")})
	} else {
		log.Printf("Login failed for user: %s", credentials.Username)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": h.I18n.GetTranslation(currentLang, "invalid_credentials")})
	}
}

// DashboardHandler displays system information after login.
func (h *Handler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)

	data, err := h.Store.GetDashboardData()
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		log.Printf(h.I18n.GetTranslation(currentLang, "error_getting_site_data")+ ": %v", err) // Translated
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
	pages, err := h.Store.GetAllPages(currentLang, h.I18n.DefaultLanguage())
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting all pages: %v", err)
		return
	}

	data := AdminPagesTemplateData{
		Title:       h.I18n.GetTranslation(currentLang, "pages_menu"),
		Pages:       pages,
		CSRFToken:   csrf.Token(r),
		CurrentPath: r.URL.Path,
		CurrentLang: currentLang,
	}

	h.renderTemplate(w, r, "admin/admin_pages.html", data)
}

// AdminEditPageHandler handles the display of the admin page edit form.
func (h *Handler) AdminEditPageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageName := vars["name"]
	
	// The language to edit can be passed as a query param, e.g., /admin/pages/edit/about?lang=ja
	// If not provided, it defaults to the user's current language preference.
	editLang := r.URL.Query().Get("lang")
	if editLang == "" {
		editLang = h.getLanguage(r)
	}

	// We fetch the page content for the specific language we want to edit.
	// The fallback logic in GetPageData ensures we get *some* content if the specific language doesn't exist yet.
	page, err := h.Store.GetPageData(pageName, editLang, h.I18n.DefaultLanguage())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.NotFound(w, r)
		} else {
			http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "internal_server_error"), http.StatusInternalServerError)
		}
		return
	}

	// Store the language code of the content that was actually retrieved.
	// This is important if page.Content is about to be cleared for a new translation.
	originalContentLang := page.Content.LanguageCode

	// If the fetched content's language doesn't match the edit language,
	// it means we've fallen back to the default. In this case, we are creating a *new* translation,
	// so we should clear the content fields for the form.
	if page.Content.LanguageCode != editLang {
		page.Content = models.PageTranslation{
			LanguageCode: editLang, // Set the language code for the new translation
		}
	}

	data := AdminEditPageTemplateData{
		Page:                        page,
		CSRFToken:                   csrf.Token(r),
		CurrentPath:                 r.URL.Path,
		CurrentLang:                 h.getLanguage(r), // This is for the UI, not the content language
		Message:                     "", // No message on initial load
		IsNew:                       false,
		SupportedLanguages:          h.I18n.GetAvailableLanguages(),
		EditLang:                    editLang,
		OriginalContentLanguageCode: originalContentLang, // Pass the original content language
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
		http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "failed_to_log_out"), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/login", http.StatusFound) // Redirect to login page
}

// ImageUploadHandler handles image uploads for the editor.
func (h *Handler) ImageUploadHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB
		http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "unable_to_parse_form"), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("image")
	if err != nil {
					http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "unable_to_get_image_from_form"), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate a random filename
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
					http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "failed_to_generate_random_filename"), http.StatusInternalServerError)
		return
	}
	filename := fmt.Sprintf("%x%s", randomBytes, filepath.Ext(handler.Filename))

	uploadDir := filepath.Join("data", "uploads")
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err = os.MkdirAll(uploadDir, 0755) // Create directory with read/write/execute permissions for owner, read/execute for others
		if err != nil {
			http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "unable_to_create_upload_directory"), http.StatusInternalServerError)
			return
		}
	}

	// Create the file
	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
					http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "unable_to_create_file_for_writing"), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination file
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "unable_to_save_file"), http.StatusInternalServerError)
		return
	}

	// Return the URL of the uploaded file
	json.NewEncoder(w).Encode(map[string]string{
		"url": "/uploads/" + filename,
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
	editLang := r.URL.Query().Get("lang")
	if editLang == "" {
		editLang = currentLang
	}

	data := AdminEditPageTemplateData{
		Page:        &models.Page{Content: models.PageTranslation{LanguageCode: editLang}},
		CSRFToken:   csrf.Token(r),
		CurrentPath: r.URL.Path,
		CurrentLang: currentLang,
		Message:     "",
		IsNew:       true,
		SupportedLanguages: h.I18n.GetAvailableLanguages(),
		EditLang:           editLang,
	}

	h.renderTemplate(w, r, "admin/admin_edit.html", data)
}

func (h *Handler) createPagePost(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)

	// The request body should contain the page shell and the content for the first translation.
	var newPage models.Page
	if err := json.NewDecoder(r.Body).Decode(&newPage); err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "invalid_request_body"), http.StatusBadRequest)
		return
	}

	// Basic validation
	if strings.TrimSpace(newPage.Name) == "" || strings.TrimSpace(newPage.Content.Title) == "" {
		http.Error(w, h.I18n.GetTranslation(currentLang, "page_name_title_required"), http.StatusBadRequest)
		return
	}

	// Set the language code for the first translation if not provided
	if newPage.Content.LanguageCode == "" {
		newPage.Content.LanguageCode = currentLang
	}

	// Check if page with the same name already exists
	_, err := h.Store.GetPageData(newPage.Name, currentLang, h.I18n.DefaultLanguage())
	if err == nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "page_exists"), http.StatusConflict)
		return
	}
	if err != gorm.ErrRecordNotFound {
		// A different error occurred
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error checking for existing page: %v", err)
		return
	}

	// Create the page and its first translation
	if err := h.Store.CreatePage(&newPage); err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error creating page: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newPage)
}

// DeletePageHandler handles the deletion of a page and all its translations.
func (h *Handler) DeletePageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageName := vars["name"]
	currentLang := h.getLanguage(r)

	err := h.Store.DeletePage(pageName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, h.I18n.GetTranslation(currentLang, "page_not_found"), http.StatusNotFound)
		} else {
			http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		}
		log.Printf("Error deleting page %s: %v", pageName, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// TranslateRequest represents the request body for the translation API.
type TranslateRequest struct {
	PageName       string `json:"page_name"` // Add PageName to identify the page
	SourceLanguage string `json:"source_language"`
	TargetLanguage string `json:"target_language"`
	Content        string `json:"content"`
	Field          string `json:"field"` // Optional: for context-specific translation
}

// TranslateResponse represents the response body for the translation API.
type TranslateResponse struct {
	TranslatedText string `json:"translated_text"`
	Error          string `json:"error,omitempty"`
}

// TranslateHandler handles translation requests.
func (h *Handler) TranslateHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)

	var req TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "invalid_request_body"), http.StatusBadRequest)
		return
	}

	// Basic validation
	if req.SourceLanguage == "" || req.TargetLanguage == "" || req.PageName == "" {
		http.Error(w, h.I18n.GetTranslation(currentLang, "invalid_request_body"), http.StatusBadRequest)
		return
	}

	contentToTranslate := req.Content

	// If content is empty, try to fetch it from the source language of the page
	if strings.TrimSpace(contentToTranslate) == "" {
		page, err := h.Store.GetPageData(req.PageName, req.SourceLanguage, h.I18n.DefaultLanguage())
		if err != nil {
			// If the source page/translation is not found, we can't translate from it.
			// Return an empty string or an error, depending on desired behavior.
			log.Printf("TranslateHandler: Could not fetch source content for page %s in lang %s: %v", req.PageName, req.SourceLanguage, err)
			http.Error(w, h.I18n.GetTranslation(currentLang, "source_content_not_found"), http.StatusNotFound)
			return
		}

		switch req.Field {
		case "title":
			contentToTranslate = page.Content.Title
		case "description":
			contentToTranslate = page.Content.Description
		case "message":
			contentToTranslate = string(page.Content.Message)
		default:
			// Unknown field, cannot fetch content
			log.Printf("TranslateHandler: Unknown field %s for content fetching", req.Field)
			http.Error(w, h.I18n.GetTranslation(currentLang, "invalid_field_for_translation"), http.StatusBadRequest)
			return
		}
	}

	translatedText, err := h.API_Translator.TranslateText(r.Context(), contentToTranslate, req.SourceLanguage, req.TargetLanguage)
	if err != nil {
		log.Printf("TranslateHandler: Error translating text: %v", err)
		http.Error(w, h.I18n.GetTranslation(currentLang, "translation_failed"), http.StatusInternalServerError)
		return
	}

	resp := TranslateResponse{
		TranslatedText: translatedText,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// getLanguage determines the language based on Accept-Language header or a cookie.
func (h *Handler) getLanguage(r *http.Request) string {
	// Check for language cookie first
	if cookie, err := r.Cookie("lang"); err == nil {
		if h.I18n.IsValidLanguage(cookie.Value) {
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
			h.DebugLog("Parsed language: %s, IsValid: %t", lang, h.I18n.IsValidLanguage(lang))
			if h.I18n.IsValidLanguage(lang) {
				return lang
			}
		}
	}

	return h.I18n.DefaultLanguage() // Fallback to default language
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
	// Example: "/about" -> "about.html"
	return strings.TrimPrefix(path, "/") + ".html"
}

// TemplateManagementData holds the data for the template management page.
type TemplateManagementData struct {
	CSRFToken     string
	CurrentPath   string
	CurrentLang   string
	Templates     []string
	StaticFiles   []string
}

// TemplateEditorData holds the data for the template editor page.
type TemplateEditorData struct {
	CSRFToken   string
	CurrentPath string
	CurrentLang string
	Title       string
	FilePath    string
	FileContent string
	FileType    string // "template" or "static"
	Message     template.HTML // Message for success/error feedback
	MessageType string // "success" or "error"
}

// TemplatePreviewData holds data for the template preview page.
type TemplatePreviewData struct {
	CSRFToken   string
	CurrentLang string
	Site        *models.Site
}

// AdminTemplatesView handles the display of the template and static file editor.
func (h *Handler) AdminTemplatesView(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)

	// --- List Template Files (non-recursive) ---
	templatesDir := "templates"
	templateFiles := []string{}
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error reading templates directory: %v", err)
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			templateFiles = append(templateFiles, entry.Name())
		}
	}

	// --- List Static Files (recursive) ---
	staticDir := "static"
	staticFiles := []string{}
	err = filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// Make the path relative to the 'static' directory for cleaner display
			relPath, err := filepath.Rel(staticDir, path)
			if err != nil {
				return err
			}
			staticFiles = append(staticFiles, filepath.ToSlash(relPath))
		}
		return nil
	})
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error walking static directory: %v", err)
		return
	}

		data := TemplateManagementData{
		CSRFToken:   csrf.Token(r),
		CurrentPath: r.URL.Path,
		CurrentLang: currentLang,
		Templates:   templateFiles,
		StaticFiles: staticFiles,
	}

	h.renderTemplate(w, r, "admin/admin_templates.html", data)
}

// AdminTemplateEditView handles displaying the editor for a single file.
func (h *Handler) AdminTemplateEditView(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)
	filePath := r.URL.Query().Get("file")
	fileType := r.URL.Query().Get("type") // "template" or "static"

	if filePath == "" || (fileType != "template" && fileType != "static") {
		http.Error(w, "Invalid request. 'file' and 'type' query parameters are required.", http.StatusBadRequest)
		return
	}

	var fullPath string
	if fileType == "template" {
		fullPath = filepath.Join("templates", filePath)
	} else {
		fullPath = filepath.Join("static", filePath)
	}

	// Security check: Ensure the path is clean and within the allowed directories.
	cleanPath := filepath.Clean(fullPath)
	if (fileType == "template" && !strings.HasPrefix(cleanPath, "templates"+string(filepath.Separator))) ||
	   (fileType == "static" && !strings.HasPrefix(cleanPath, "static"+string(filepath.Separator))) {
		http.Error(w, "Access denied: Path is outside of the allowed directories.", http.StatusForbidden)
		return
	}
	
	// Security check 2: Prevent reading directories
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Error accessing file.", http.StatusInternalServerError)
		}
		return
	}
	if info.IsDir() {
		http.Error(w, "Cannot edit a directory.", http.StatusBadRequest)
		return
	}


	content, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error reading file %s: %v", cleanPath, err)
		return
	}

	data := TemplateEditorData{
		Title:       h.I18n.GetTranslation(currentLang, "edit_file_title") + " " + filePath,
		CSRFToken:   csrf.Token(r),
		CurrentPath: r.URL.Path,
		CurrentLang: currentLang,
		FilePath:    filePath,
		FileContent: string(content),
		FileType:    fileType,
	}

	h.renderTemplate(w, r, "admin/admin_template_edit.html", data)
}

// AdminTemplateUpdate handles saving the updated file content.
func (h *Handler) AdminTemplateUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	filePath := r.FormValue("filePath")
	fileContent := r.FormValue("fileContent")
	fileType := r.FormValue("fileType")

	if filePath == "" || (fileType != "template" && fileType != "static") {
		http.Error(w, "Invalid request. 'filePath' and 'fileType' are required.", http.StatusBadRequest)
		return
	}

	var fullPath string
	if fileType == "template" {
		fullPath = filepath.Join("templates", filePath)
	} else {
		fullPath = filepath.Join("static", filePath)
	}

	// Security check: Ensure the path is clean and within the allowed directories.
	cleanPath := filepath.Clean(fullPath)
	if (fileType == "template" && !strings.HasPrefix(cleanPath, "templates"+string(filepath.Separator))) ||
	   (fileType == "static" && !strings.HasPrefix(cleanPath, "static"+string(filepath.Separator))) {
		http.Error(w, "Access denied: Path is outside of the allowed directories.", http.StatusForbidden)
		return
	}
	
	// Security check 2: Prevent writing to directories
	info, err := os.Stat(cleanPath)
	if err != nil && !os.IsNotExist(err) { // If file doesn't exist, it's fine, but other errors are bad
		http.Error(w, "Error accessing file.", http.StatusInternalServerError)
		log.Printf("Error statting file before write %s: %v", cleanPath, err)
		return
	}
	if info != nil && info.IsDir() {
		http.Error(w, "Cannot write to a directory.", http.StatusBadRequest)
		return
	}

	if err := os.WriteFile(cleanPath, []byte(fileContent), 0644); err != nil {
		log.Printf("Error writing to file %s: %v", cleanPath, err)
		http.Redirect(w, r, fmt.Sprintf("%s?file=%s&type=%s&message=%s&messageType=error", r.URL.Path, filePath, fileType, h.I18n.GetTranslation(h.getLanguage(r), "admin.file_save_failed")), http.StatusFound)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("%s?file=%s&type=%s&message=%s&messageType=success", r.URL.Path, filePath, fileType, h.I18n.GetTranslation(h.getLanguage(r), "admin.file_save_success")), http.StatusFound)
}

// AdminTemplatePreview handles rendering a preview of a template file.
func (h *Handler) AdminTemplatePreview(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		http.Error(w, "Missing 'file' query parameter.", http.StatusBadRequest)
		return
	}

	// Security check
	cleanPath := filepath.Clean(filepath.Join("templates", filePath))
	if !strings.HasPrefix(cleanPath, "templates"+string(filepath.Separator)) {
		http.Error(w, "Access denied: Template is outside of the allowed directory.", http.StatusForbidden)
		return
	}
	
	// This is a simplified preview. It parses the requested template along with the base layouts.
	// It won't have access to the full context of other templates, but it's good for a direct preview.
	// It uses a nil data object, so templates expecting data may show errors.
	
	currentLang := h.getLanguage(r)
	
	// We need to parse the base templates along with the specific template file.
	// This mimics how the main renderTemplate function works but for a single, dynamic file.
	funcMap := template.FuncMap{
		"T": func(key string) string {
			return h.I18n.GetTranslation(currentLang, key)
		},
	}

	tmpl, err := template.New("preview").Funcs(funcMap).ParseFiles(
		"templates/base.html",
		"templates/header.html",
		"templates/footer.html",
		"templates/nav.html",
		cleanPath, // The actual template to preview
	)

	if err != nil {
		http.Error(w, "Error parsing template for preview.", http.StatusInternalServerError)
		log.Printf("Error parsing preview for %s: %v", cleanPath, err)
		return
	}

	// We execute "base" which should in turn call our specific template's content block.
	// We pass a nil data object.
	
	siteData, err := h.Store.GetSiteData()
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting site data for template preview: %v", err)
		return
	}

	data := TemplatePreviewData{
		CSRFToken:   csrf.Token(r),
		CurrentLang: currentLang,
		Site:        siteData,
	}

	err = tmpl.ExecuteTemplate(w, "base", data)
	if err != nil {
		http.Error(w, "Error executing template for preview.", http.StatusInternalServerError)
		log.Printf("Error executing preview for %s: %v", cleanPath, err)
		return
	}
}
