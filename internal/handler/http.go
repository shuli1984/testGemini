package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"gemini-demo/internal/auth"
	"gemini-demo/internal/config"
	"gemini-demo/internal/i18n"
	"gemini-demo/internal/logger"
	"gemini-demo/internal/models"
	"gemini-demo/internal/translator"
	"gemini-demo/internal/util"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/shirou/gopsutil/cpu"
	"gorm.io/gorm"
)

type Handler struct {
	Cfg            *config.Config
	Store          models.DataStore
	AuthService    *auth.AuthService
	Templates      map[string]*template.Template // Changed to map
	DebugLog       func(format string, v ...interface{})
	I18n           *i18n.Translator
	API_Translator translator.Translator
	DebugMode      bool
	ErrorLogger    *logger.InMemoryLogCollector
	StartTime      time.Time
}

// LogError logs an error to the standard logger and the in-memory collector.
func (h *Handler) LogError(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf(msg)
	if h.ErrorLogger != nil {
		h.ErrorLogger.Add(msg)
	}
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
	CSRFToken     string
	CurrentPath   string
	CurrentLang   string
	PageCount     int64
	TemplateCount int
	RecentPages   []models.Page
	GoVersion     string
	OS            string
	Arch          string
	Uptime        string
	MemoryUsage   string
	NumGoroutine  int
	CPUUsage      string
	RecentErrors  []string
	RecentLoginLogs []models.LoginLog
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
	SiteDefaultLanguage         string
}

// AdminPagesTemplateData holds data for the admin pages list template.
type AdminPagesTemplateData struct {
	Pages       []models.Page
	CSRFToken   string
	CurrentPath string
	CurrentLang string
	Title       string
}

// AdminSettingsTemplateData holds data for the admin settings page.
type AdminSettingsTemplateData struct {
	CSRFToken          string
	Settings           *config.SiteConfig
	Message            string
	CurrentLang        string // UI language
	CurrentPath        string
	NavigationJSON     string
	SupportedLanguages []string // For language switcher
	EditLang           string   // Language being edited
}

// PageCombinedData holds data for a page template, combining page and site data.
type PageCombinedData struct {
	Page       *models.Page
	SiteConfig *config.SiteConfig
	CurrentLang string
}

// IndexTemplateData holds data for the index page template.
type IndexTemplateData struct {
	SiteConfig    *config.SiteConfig
	CarouselItems []models.CarouselItem
	CoreSolutions []models.Page
	CurrentLang   string
	Page          *models.Page // Add Page field for header compatibility
}

// PagesListTemplateData holds data for the pages list page template.
type PagesListTemplateData struct {
	Page       *models.Page // Add Page field for header compatibility
	Pages      []models.Page
	SiteConfig *config.SiteConfig
	CurrentLang string
	ContentTemplateName string
}

func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)
	siteConfig, err := h.Store.GetSiteConfig(currentLang, h.I18n.DefaultLanguage())
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting site config: %v", err)
		return
	}

	h.DebugLog("IndexHandler: currentLang = %s", currentLang)
	siteConfigJSON, _ := json.Marshal(siteConfig)
	h.DebugLog("IndexHandler: siteConfig = %s", string(siteConfigJSON))

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
			BackgroundImage: "/static/images/hero-2.jpg", // Placeholder image
		},
		{
			Title:         template.HTML(h.I18n.GetTranslation(currentLang, "common.carousel_title_3")),
			Description:   template.HTML(h.I18n.GetTranslation(currentLang, "common.carousel_description_3")),
			ButtonText:    h.I18n.GetTranslation(currentLang, "common.carousel_button_text_3"),
			ButtonLink:    "#about",
			BackgroundImage: "/static/images/hero-3.jpg", // Placeholder image
		},
	}

	coreSolutions, err := h.Store.GetCoreSolutions(currentLang, h.I18n.DefaultLanguage())
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(h.getLanguage(r), "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting core solutions: %v", err)
		return
	}

	data := IndexTemplateData{
		SiteConfig:    siteConfig,
		CarouselItems: carouselItems,
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

	siteConfig, err := h.Store.GetSiteConfig(currentLang, h.I18n.DefaultLanguage())
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		log.Printf("Error getting site config: %v", err)
		return
	}

	data := PagesListTemplateData{
		Pages:      pages,
		SiteConfig: siteConfig,
		CurrentLang: currentLang,
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

	siteConfig, err := h.Store.GetSiteConfig(currentLang, h.I18n.DefaultLanguage())
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError) // Translated
		log.Printf("error getting site data: %v", err) // Translated
		return
	}

	combinedData := PageCombinedData{
		Page:       pageData,
		SiteConfig: siteConfig,
		CurrentLang: currentLang,
	}

	h.renderTemplate(w, r, getTemplateName(r), combinedData)
}

func (h *Handler) AboutHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)
	h.DebugLog("AboutHandler: currentLang = %s", currentLang) // Add this line

	siteConfig, err := h.Store.GetSiteConfig(currentLang, h.I18n.DefaultLanguage())
	if err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "internal_server_error"), http.StatusInternalServerError)
		h.DebugLog("Error getting site data for about page: %v", err)
		return
	}

	data := struct {
		SiteConfig  *config.SiteConfig
		CurrentLang string
		Page        *models.Page // Add Page field for header compatibility
	}{
		SiteConfig:  siteConfig,
		CurrentLang: currentLang,
		Page:        nil, // Initialize Page to nil for about page
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

	// Log the login attempt
	logAttempt := &models.LoginLog{
		Username:  credentials.Username,
		IPAddress: getIPAddress(r),
		UserAgent: r.UserAgent(),
	}

	if h.AuthService.Authenticate(credentials.Username, credentials.Password) {
		logAttempt.Success = true
		if err := h.Store.CreateLoginLog(logAttempt); err != nil {
			h.LogError("failed to create login log: %v", err)
		}

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
		logAttempt.Success = false
		if err := h.Store.CreateLoginLog(logAttempt); err != nil {
			h.LogError("failed to create login log: %v", err)
		}

		log.Printf("Login failed for user: %s", credentials.Username)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": h.I18n.GetTranslation(currentLang, "invalid_credentials")})
	}
}

// DashboardHandler displays system information after login.
func (h *Handler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	currentLang := h.getLanguage(r)

	// Page Count
	pageCount, err := h.Store.GetPageCount()
	if err != nil {
		h.LogError("Error getting page count for dashboard: %v", err)
		// We can still proceed, just show 0 for page count
		pageCount = 0
	}

	// Template Count
	templateFiles, err := os.ReadDir("templates")
	if err != nil {
		h.LogError("Error reading templates directory: %v", err)
	}
	templateCount := 0
	for _, file := range templateFiles {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".html") {
			templateCount++
		}
	}

	// Recent Pages
	recentPages, err := h.Store.GetRecentPages(5, currentLang, h.I18n.DefaultLanguage())
	if err != nil {
		h.LogError("Error getting recent pages for dashboard: %v", err)
	}

	// System Info
	goVersion := runtime.Version()
	osName := runtime.GOOS
	arch := runtime.GOARCH
	uptime := time.Since(h.StartTime).Round(time.Second).String()
	numGoroutine := runtime.NumGoroutine()

	// Memory Usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsage := fmt.Sprintf("%d MB", m.Alloc/1024/1024)

	// CPU Usage
	// Reduced the interval to 100ms to avoid blocking the request for a full second.
	cpuUsage := "N/A"
	percentages, err := cpu.Percent(100*time.Millisecond, false)
	if err == nil && len(percentages) > 0 {
		cpuUsage = fmt.Sprintf("%.2f%%", percentages[0])
	} else if err != nil {
		h.LogError("Error getting CPU usage: %v", err)
	}

	// Recent Errors
	recentErrors := h.ErrorLogger.Get()

	// Recent Login Logs
	recentLoginLogs, err := h.Store.GetRecentLoginLogs(10)
	if err != nil {
		h.LogError("Error getting recent login logs for dashboard: %v", err)
	}

	templateData := DashboardTemplateData{
		CSRFToken:       csrf.Token(r),
		CurrentPath:     r.URL.Path,
		CurrentLang:     currentLang,
		PageCount:       pageCount,
		TemplateCount:   templateCount,
		RecentPages:     recentPages,
		GoVersion:       goVersion,
		OS:              osName,
		Arch:            arch,
		Uptime:          uptime,
		MemoryUsage:     memoryUsage,
		NumGoroutine:    numGoroutine,
		CPUUsage:        cpuUsage,
		RecentErrors:    recentErrors,
		RecentLoginLogs: recentLoginLogs,
	}
	h.renderTemplate(w, r, "admin/admin_dashboard.html", templateData)
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
		SiteDefaultLanguage:         h.I18n.DefaultLanguage(),
	}

	h.renderTemplate(w, r, "admin/admin_edit.html", data)
}


// AdminSettingsHandler displays the settings page.
func (h *Handler) AdminSettingsHandler(w http.ResponseWriter, r *http.Request) {
	// Determine the language to edit
	editLang := r.URL.Query().Get("lang")
	if editLang == "" {
		editLang = h.I18n.DefaultLanguage()
	}

	siteConfig, err := h.Store.GetSiteConfig(editLang, h.I18n.DefaultLanguage())
	if err != nil {
		log.Printf("Error getting site config: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if siteConfig.Navigation == nil {
		siteConfig.Navigation = []config.NavigationItem{}
	}
	navJSON, err := json.Marshal(siteConfig.Navigation)
	if err != nil {
		log.Printf("Error marshalling navigation: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := AdminSettingsTemplateData{
		CSRFToken:          csrf.Token(r),
		Settings:           siteConfig,
		CurrentLang:        h.getLanguage(r), // UI language
		CurrentPath:        r.URL.Path,
		NavigationJSON:     string(navJSON),
		SupportedLanguages: h.I18n.GetAvailableLanguages(),
		EditLang:           editLang,
	}
	h.renderTemplate(w, r, "admin/admin_settings.html", data)
}

// UpdateSettingsHandler handles updating the site settings.
func (h *Handler) UpdateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// The language that was edited
	editLang := r.FormValue("editLang")
	if editLang == "" {
		// Fallback, though it should always be submitted by the form
		editLang = h.I18n.DefaultLanguage()
	}

	currentLang := h.getLanguage(r) // UI language

	// Populate a new config object from the form.
	formConfig := &config.SiteConfig{
		Title:             r.FormValue("siteTitle"),
		Tagline:           r.FormValue("siteTagline"),
		Logo:              r.FormValue("siteLogo"),
		Favicon:           r.FormValue("favicon"),
		DefaultLanguage:   r.FormValue("defaultLanguage"),
		Timezone:          r.FormValue("timezone"),
		HomePage:          r.FormValue("homePage"),
		MetaDescription:   r.FormValue("metaDescription"),
		MetaKeywords:      r.FormValue("metaKeywords"),
		GoogleAnalyticsID: r.FormValue("googleAnalyticsID"),
		MaintenanceMode:   r.FormValue("maintenanceMode") == "on",
		MaintenanceMessage: r.FormValue("maintenanceMessage"),
	}
	navJSON := r.FormValue("navigationJson")

	// --- Validation ---
	var validationErrors []string

	// Rule 1: Title is required (only for the language being edited)
	if strings.TrimSpace(formConfig.Title) == "" {
		validationErrors = append(validationErrors, h.I18n.GetTranslation(currentLang, "admin.settings.error.title_required"))
	}

	// Rule 2: Default Language must be a valid language
	if !h.I18n.IsValidLanguage(formConfig.DefaultLanguage) {
		availableLangs := strings.Join(h.I18n.GetAvailableLanguages(), ", ")
		errorMsg := fmt.Sprintf(h.I18n.GetTranslation(currentLang, "settings.error.invalid_language_format"), formConfig.DefaultLanguage, availableLangs)
		validationErrors = append(validationErrors, errorMsg)
	}

	// Rule 3: Google Analytics ID format (basic check)
	gaID := formConfig.GoogleAnalyticsID
	if gaID != "" && !strings.HasPrefix(gaID, "G-") && !strings.HasPrefix(gaID, "UA-") && !strings.HasPrefix(gaID, "GTM-") {
		validationErrors = append(validationErrors, h.I18n.GetTranslation(currentLang, "settings.error.invalid_ga_id"))
	}

	// Rule 4: Navigation JSON
	var navigation []config.NavigationItem
	if navJSON != "" {
		if err := json.Unmarshal([]byte(navJSON), &navigation); err != nil {
			validationErrors = append(validationErrors, h.I18n.GetTranslation(currentLang, "settings.error.invalid_nav_json"))
		}
	}
	formConfig.Navigation = navigation

	// --- End Validation ---

	// Prepare template data. We'll use this for all return paths.
	data := AdminSettingsTemplateData{
		CSRFToken:          csrf.Token(r),
		Settings:           formConfig, // Start with form data to preserve input on error
		CurrentLang:        currentLang,
		CurrentPath:        r.URL.Path,
		NavigationJSON:     navJSON, // Use the raw JSON string from the form
		SupportedLanguages: h.I18n.GetAvailableLanguages(),
		EditLang:           editLang,
	}

	// If there are validation errors, re-render the form with the errors and user's input.
	if len(validationErrors) > 0 {
		data.Message = strings.Join(validationErrors, "; ")
		w.WriteHeader(http.StatusBadRequest)
		h.renderTemplate(w, r, "admin/admin_settings.html", data)
		return
	}

	// If validation passes, save the config for the specific language.
	if err := h.Store.SaveSiteConfig(formConfig, editLang); err != nil {
		log.Printf("Error saving site config: %v", err)
		data.Message = h.I18n.GetTranslation(currentLang, "admin.settings_save_failed")
		w.WriteHeader(http.StatusInternalServerError)
		h.renderTemplate(w, r, "admin/admin_settings.html", data)
		return
	}

	// --- Success Path ---

	// Also need to update the global config in the running application
	// We reload it for the app's default language to ensure consistency
	// A more advanced setup might involve a config cache invalidation mechanism
	reloadedGlobalConfig, err := h.Store.GetSiteConfig(h.Cfg.Site.DefaultLanguage, h.Cfg.Site.DefaultLanguage)
	if err != nil {
		log.Printf("CRITICAL: Failed to reload site config after update: %v", err)
	} else {
		h.Cfg.Site = *reloadedGlobalConfig
	}

	// Fetch the newly saved config to display
	displayConfig, err := h.Store.GetSiteConfig(editLang, h.I18n.DefaultLanguage()) // Fetch for the edited language
	if err != nil {
		log.Printf("CRITICAL: Failed to reload site config for display: %v", err)
		// Fallback to formConfig if reload fails
		displayConfig = formConfig
	}

	// Re-marshal navigation JSON from the definitive saved data
	reloadedNavJSON, err := json.Marshal(displayConfig.Navigation)
	if err != nil {
		log.Printf("Error marshalling reloaded navigation: %v", err)
		reloadedNavJSON = []byte(navJSON) // Fallback to original form submission
	}

	// Update data for successful render
	data.Settings = displayConfig
	data.Message = h.I18n.GetTranslation(currentLang, "admin.settings_saved_successfully")
	data.NavigationJSON = string(reloadedNavJSON)

	h.renderTemplate(w, r, "admin/admin_settings.html", data)
}



// MaintenanceMiddleware checks if the site is in maintenance mode.
func (h *Handler) MaintenanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bypass maintenance mode for admin, login, and static assets
		isAdminRoute := strings.HasPrefix(r.URL.Path, "/admin")
		isLoginRoute := r.URL.Path == "/login"
		isStatic := strings.HasPrefix(r.URL.Path, h.Cfg.Static.URLPrefix)

		// Check if maintenance mode is enabled and the route is not exempt
		if h.Cfg.Site.MaintenanceMode && !isAdminRoute && !isLoginRoute && !isStatic {

			// Prepare data for the maintenance template
			data := struct {
				Site        *config.SiteConfig
				CurrentLang string
			}{
				Site:        &h.Cfg.Site,
				CurrentLang: h.getLanguage(r),
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)

			// Manually parse and execute the maintenance template to avoid the site's base layout
			// Create a FuncMap for the 'T' function
			funcMap := template.FuncMap{
				"T": func(lang, key string) string {
					return h.I18n.GetTranslation(lang, key)
				},
			}

			tmpl, err := template.New("maintenance.html").Funcs(funcMap).ParseFiles("templates/maintenance.html")
			if err != nil {
				log.Printf("Error parsing maintenance template: %v", err)
				http.Error(w, "Error displaying maintenance page.", http.StatusInternalServerError)
				return
			}

			err = tmpl.Execute(w, data)
			if err != nil {
				log.Printf("Error executing maintenance template: %v", err)
				// http.Error is already sent by this point, so just log
			}
			return
		}

		next.ServeHTTP(w, r)
	})
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

	var payload struct {
		Name           string        `json:"Name"`
		Title          string        `json:"Title"`
		Description    string        `json:"Description"`
		Message        template.HTML `json:"Message"`
		IsCoreSolution bool          `json:"IsCoreSolution"`
		Icon           string        `json:"Icon"`
		LanguageCode   string        `json:"LanguageCode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, h.I18n.GetTranslation(currentLang, "invalid_request_body"), http.StatusBadRequest)
		return
	}

	// Basic validation
	if strings.TrimSpace(payload.Name) == "" || strings.TrimSpace(payload.Title) == "" {
		http.Error(w, h.I18n.GetTranslation(currentLang, "page_name_title_required"), http.StatusBadRequest)
		return
	}

	langCode := payload.LanguageCode
	if langCode == "" {
		langCode = currentLang
	}

	newPage := &models.Page{
		Name:           payload.Name,
		IsCoreSolution: payload.IsCoreSolution,
		Icon:           payload.Icon,
		Content: models.PageTranslation{
			Title:        payload.Title,
			Description:  payload.Description,
			Message:      payload.Message,
			LanguageCode: langCode,
		},
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
	if err := h.Store.CreatePage(newPage); err != nil {
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
	PageName       string `json:"page_name,omitempty"` // Optional: to identify the page
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
	if req.SourceLanguage == "" || req.TargetLanguage == "" {
		http.Error(w, h.I18n.GetTranslation(currentLang, "invalid_request_body"), http.StatusBadRequest)
		return
	}

	contentToTranslate := req.Content

	// If content is empty, try to fetch it from the source language of the page or setting
	if strings.TrimSpace(contentToTranslate) == "" {
		if req.PageName != "" {
			// It's a page translation
			page, err := h.Store.GetPageData(req.PageName, req.SourceLanguage, h.I18n.DefaultLanguage())
			if err != nil {
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
				log.Printf("TranslateHandler: Unknown page field %s for content fetching", req.Field)
				http.Error(w, h.I18n.GetTranslation(currentLang, "invalid_field_for_translation"), http.StatusBadRequest)
				return
			}
		} else if req.Field != "" {
			// It's a setting translation
			settingValue, err := h.Store.GetSettingValue(req.Field, req.SourceLanguage)
			if err != nil {
				log.Printf("TranslateHandler: Could not fetch source content for setting %s in lang %s: %v", req.Field, req.SourceLanguage, err)
				http.Error(w, h.I18n.GetTranslation(currentLang, "source_content_not_found"), http.StatusNotFound)
				return
			}
			contentToTranslate = settingValue
		} else {
			// If we have neither PageName nor Field, we don't know what to fetch.
			http.Error(w, h.I18n.GetTranslation(currentLang, "invalid_request_body"), http.StatusBadRequest)
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

// getIPAddress extracts the user's IP address from the request.
func getIPAddress(r *http.Request) string {
    // Check for X-Forwarded-For header first (for proxies)
    forwarded := r.Header.Get("X-Forwarded-For")
    if forwarded != "" {
        // X-Forwarded-For can be a comma-separated list of IPs. The first one is the original client.
        ips := strings.Split(forwarded, ",")
        return strings.TrimSpace(ips[0])
    }

    // Fallback to RemoteAddr
    ip, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        // If splitting fails, RemoteAddr might be just the IP, which is fine.
        return r.RemoteAddr
    }
    return ip
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
	Site        *config.SiteConfig
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
	
	siteData, err := h.Store.GetSiteConfig(currentLang, h.I18n.DefaultLanguage())
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