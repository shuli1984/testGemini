package server

import (
	"fmt"
	"gemini-demo/internal/auth"
	"gemini-demo/internal/config"
	"gemini-demo/internal/handler"
	"gemini-demo/internal/i18n"
	"gemini-demo/internal/models"
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// DebugLog is a placeholder for the debug logging function from main.go
var DebugLog func(format string, v ...interface{})

// New creates a new HTTP server with configured routes and handlers.
// This function is the entry point for the server.
func New(cfg *config.Config, db *gorm.DB, tmpl *template.Template, csrfMiddleware func(http.Handler) http.Handler, translator *i18n.Translator) *http.Server {
	r := mux.NewRouter()

	// Serve static files
	staticFileServer := http.FileServer(http.Dir(cfg.Static.Dir))
	r.PathPrefix(cfg.Static.URLPrefix).Handler(http.StripPrefix(cfg.Static.URLPrefix, staticFileServer))

	authService := auth.NewAuthService(cfg.Auth.SessionKey)

	// Create the DataStore implementation
	dbStore := &models.DBStore{DB: db}

	// Create an empty FuncMap for templates.
	funcMap := template.FuncMap{}

	clonedTemplates, err := tmpl.Clone()
	if err != nil {
		panic(fmt.Sprintf("Failed to clone templates: %v", err))
	}
	clonedTemplates = clonedTemplates.Funcs(funcMap)

	// Pass the parsed templates to the handler
	h := &handler.Handler{Store: dbStore, AuthService: authService, Templates: clonedTemplates, DebugLog: DebugLog, Translator: translator}

	handlers := map[string]http.HandlerFunc{
		"IndexHandler":         h.IndexHandler,
		"AboutHandler":         h.AboutHandler,
		"PageHandler":          h.PageHandler,
		"UpdatePageHandler":    h.UpdatePageHandler,
		"LoginHandler":         h.LoginHandler,
		"DashboardHandler":     h.DashboardHandler,
		"AdminRedirectHandler": h.AdminRedirectHandler,
		"LogoutHandler":        h.LogoutHandler,
		"PagesHandler":         h.PagesHandler,
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
			DebugLog("Server: Before CSRF - Request URL: %s, Method: %s, Headers: %v", req.URL.Path, req.Method, req.Header)
			csrfMiddleware(r).ServeHTTP(w, req)
			DebugLog("Server: After CSRF - Request processed for URL: %s", req.URL.Path)
		})
	}

	return &http.Server{
		Handler: finalHandler,
	}
}
