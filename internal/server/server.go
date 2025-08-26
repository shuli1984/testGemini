package server

import (
	"gemini-demo/internal/auth"
	"gemini-demo/internal/handler"
	"net/http"
	"html/template"

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

// New creates a new HTTP server with configured routes and handlers.
func New(db *gorm.DB, tmpl *template.Template, csrfMiddleware func(http.Handler) http.Handler) *http.Server { // Added csrfMiddleware argument
	r := mux.NewRouter()

	// Serve static files
	staticURLPrefix := viper.GetString("static.url_prefix")
	staticDir := viper.GetString("static.dir")
	staticFileServer := http.FileServer(http.Dir(staticDir))
	r.PathPrefix(staticURLPrefix).Handler(http.StripPrefix(staticURLPrefix, staticFileServer))

	authService := auth.NewAuthService()
	// Pass the parsed templates to the handler
	h := &handler.Handler{DB: db, AuthService: authService, Templates: tmpl}

	handlers := map[string]http.HandlerFunc{
		"IndexHandler":         h.IndexHandler,
		"AboutHandler":         h.AboutHandler,
		"PageHandler":          h.PageHandler,
		"UpdatePageHandler":    h.UpdatePageHandler,
		"LoginHandler":         h.LoginHandler,
		"DashboardHandler":     h.DashboardHandler,
		"AdminRedirectHandler": h.AdminRedirectHandler,
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
	return &http.Server{
		Handler: csrfMiddleware(r), // Apply CSRF middleware here
	}
}
