package server

import (
	"gemini-demo/internal/auth"
	"gemini-demo/internal/handler"
	"net/http"

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

func New(db *gorm.DB) *http.Server {
	r := mux.NewRouter()

	// Serve static files
	staticURLPrefix := viper.GetString("static.url_prefix")
	staticDir := viper.GetString("static.dir")
	staticFileServer := http.FileServer(http.Dir(staticDir))
	r.PathPrefix(staticURLPrefix).Handler(http.StripPrefix(staticURLPrefix, staticFileServer))

	authService := auth.NewAuthService()
	h := &handler.Handler{DB: db, AuthService: authService} // Create an instance of the handler

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

	return &http.Server{
		Handler: r,
	}
}
