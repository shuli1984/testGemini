package server

import (
	"fmt" // Added for fmt.Sprintf
	"gemini-demo/internal/auth"
	"gemini-demo/internal/handler"
	"gemini-demo/internal/i18n" // New import for i18n
	"net/http"
	"html/template"
	// "log" // Remove this import

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Route struct {
	Path         string   `mapstructure:"path"`
	Handler      string   `mapstructure:"handler"`
	Methods      []string `mapstructure:"methods"`
	AuthRequired bool     `mapstructure:"auth_required"`
}

// DebugLog is a placeholder for the debug logging function from main.go
// In a real application, you would pass a logger instance or use a global logger.
// For this exercise, we'll assume DebugLog is accessible.
var DebugLog func(format string, v ...interface{})

// New creates a new HTTP server with configured routes and handlers.
func New(db *gorm.DB, tmpl *template.Template, csrfMiddleware func(http.Handler) http.Handler, translator *i18n.Translator) *http.Server { // Added translator argument
	r := mux.NewRouter()

	// Serve static files
	staticURLPrefix := viper.GetString("static.url_prefix")
	staticDir := viper.GetString("static.dir")
	staticFileServer := http.FileServer(http.Dir(staticDir))
	r.PathPrefix(staticURLPrefix).Handler(http.StripPrefix(staticURLPrefix, staticFileServer))

	authService := auth.NewAuthService()

	// Create an empty FuncMap for templates. The 'T' function will be passed directly in the template data.
	funcMap := template.FuncMap{}

	// Clone the template set and add the FuncMap
	// This ensures that each handler gets a template set with the correct functions
	// and avoids modifying the global template set.
	clonedTemplates, err := tmpl.Clone()
	if err != nil {
		// Handle error, perhaps log and panic or return an error
		panic(fmt.Sprintf("Failed to clone templates: %v", err))
	}
	clonedTemplates = clonedTemplates.Funcs(funcMap)


	// Pass the parsed templates to the handler
	h := &handler.Handler{DB: db, AuthService: authService, Templates: clonedTemplates, DebugLog: DebugLog, Translator: translator}

	handlers := map[string]http.HandlerFunc{
		"IndexHandler":         h.IndexHandler,
		"AboutHandler":         h.AboutHandler,
		"PageHandler":          h.PageHandler,
		"UpdatePageHandler":    h.UpdatePageHandler,
		"LoginHandler":         h.LoginHandler,
		"DashboardHandler":     h.DashboardHandler,
		"AdminRedirectHandler": h.AdminRedirectHandler,
		"LogoutHandler":        h.LogoutHandler, // Added LogoutHandler
	}

	var routes []Route
	if err := viper.UnmarshalKey("routes", &routes); err != nil {
		panic(err)
	}

	for _, route := range routes {
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
			// Attempt to get CSRF token from header (common for AJAX)
			csrfTokenHeader := req.Header.Get("X-CSRF-Token")
			if csrfTokenHeader != "" {
				DebugLog("Server: Before CSRF - X-CSRF-Token Header: %s", csrfTokenHeader)
			}
			// Attempt to get CSRF token from form (common for form submissions)
			if req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodDelete {
				if err := req.ParseForm(); err == nil {
					csrfTokenForm := req.Form.Get("csrf_token") // Assuming 'csrf_token' is the form field name
					if csrfTokenForm != "" {
						DebugLog("Server: Before CSRF - csrf_token Form Field: %s", csrfTokenForm)
					}
				}
			}

			// Pass the request to the actual CSRF middleware
			csrfMiddleware(r).ServeHTTP(w, req)

			// Log after CSRF (if control returns here, it means CSRF didn't block it immediately)
			DebugLog("Server: After CSRF - Request processed for URL: %s", req.URL.Path)
		})
	}

	return &http.Server{
		Handler: finalHandler,
	}
}