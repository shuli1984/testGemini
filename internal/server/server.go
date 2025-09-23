package server

import (
	"gemini-demo/internal/auth"
	"gemini-demo/internal/config"
	"gemini-demo/internal/handler"
	"gemini-demo/internal/i18n"
	"gemini-demo/internal/logger"
	"gemini-demo/internal/models"
	"gemini-demo/internal/translator"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// DebugLog is a placeholder for the debug logging function from main.go
// New creates a new HTTP server with configured routes and handlers.
// This function is the entry point for the server.
func New(cfg *config.Config, db *gorm.DB, tmpl map[string]*template.Template, csrfMiddleware func(http.Handler) http.Handler, i18nTranslator *i18n.Translator, apiTranslator translator.Translator, debugLog func(format string, v ...interface{}), debugMode bool, errorLogger *logger.InMemoryLogCollector, startTime time.Time) *http.Server {
	r := mux.NewRouter()

	// Serve static files
	staticFileServer := http.FileServer(http.Dir(cfg.Static.Dir))
	r.PathPrefix(cfg.Static.URLPrefix).Handler(http.StripPrefix(cfg.Static.URLPrefix, staticFileServer))

	// Serve uploaded files from data/uploads
	uploadFileServer := http.FileServer(http.Dir("data/uploads"))
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", uploadFileServer))

	authService := auth.NewAuthService(cfg.Auth.SessionKey)

	// Create the DataStore implementation
	dbStore := &models.DBStore{DB: db}

	// Pass the parsed templates to the handler
	h := &handler.Handler{Cfg: cfg, Store: dbStore, AuthService: authService, Templates: tmpl, DebugLog: debugLog, I18n: i18nTranslator, API_Translator: apiTranslator, DebugMode: debugMode, ErrorLogger: errorLogger, StartTime: startTime}

	handlers := map[string]http.HandlerFunc{
		"IndexHandler":         h.IndexHandler,
		"AboutHandler":         h.AboutHandler,
		"PageHandler":          h.PageHandler,
		"PagesListHandler":     h.PagesListHandler,
		"UpdatePageHandler":    h.UpdatePageHandler,
		"LoginHandler":         h.LoginHandler,
		"DashboardHandler":     h.DashboardHandler,
		"AdminRedirectHandler": h.AdminRedirectHandler,
		"LogoutHandler":        h.LogoutHandler,
		"PagesHandler":         h.PagesHandler,
		"AdminEditPageHandler": h.AdminEditPageHandler,
		"AdminNewPageHandler":  h.AdminNewPageHandler,
		"DeletePageHandler":    h.DeletePageHandler,
		"TranslateHandler":     h.TranslateHandler,
		"ImageUploadHandler":   h.ImageUploadHandler,
		"AdminTemplatesView":    h.AdminTemplatesView,
		"AdminTemplateEditView": h.AdminTemplateEditView,
		"AdminTemplateUpdate":   h.AdminTemplateUpdate,
		"AdminTemplatePreview":  h.AdminTemplatePreview,
		"AdminSettingsHandler":  h.AdminSettingsHandler,
		"UpdateSettingsHandler": h.UpdateSettingsHandler,
	}

	for _, route := range cfg.Routes {
		if handlerFunc, ok := handlers[route.Handler]; ok {
			var finalHandler http.Handler = handlerFunc
			if route.AuthRequired {
				finalHandler = authService.Middleware(handlerFunc)
			}
			r.Handle(route.Path, finalHandler).Methods(route.Methods...)
		}
	}

	// Apply CSRF middleware to the main router
	var finalHandler http.Handler = r
	if csrfMiddleware != nil {
		finalHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			debugLog("Server: Before CSRF - Request URL: %s, Method: %s, Headers: %v", req.URL.Path, req.Method, req.Header)
			csrfMiddleware(r).ServeHTTP(w, req)
			debugLog("Server: After CSRF - Request processed for URL: %s", req.URL.Path)
		})
	}

	finalHandler = h.MaintenanceMiddleware(finalHandler)

	return &http.Server{
		Handler: finalHandler,
	}
}
