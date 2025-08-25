package auth

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
)

const (
	sessionName = "gemini-session"
	sessionKey  = "user_logged_in"
)

// Store will hold the session store
var Store sessions.Store

// AuthService provides authentication services.
type AuthService struct {
	store sessions.Store
}

var OsExit = os.Exit
var FatalLogger = log.Fatal

// NewAuthService creates a new AuthService using the global viper instance.
func NewAuthService() *AuthService {
	return NewAuthServiceWithViper(viper.GetViper())
}

// NewAuthServiceWithViper creates a new AuthService with a specific viper instance.
func NewAuthServiceWithViper(vp *viper.Viper) *AuthService {
	// In a production environment, use a more robust key management system.
	// The key should be stored securely and not hardcoded.
	key := vp.GetString("auth.session_key")
	if key == "" {
		FatalLogger("Session key not found in config. Please set auth.session_key")
	}
	Store = sessions.NewCookieStore([]byte(key))
	return &AuthService{store: Store}
}

// Authenticate checks user credentials.
// In a real application, this would check against a database.
func (s *AuthService) Authenticate(username, password string) bool {
	expectedUsername := os.Getenv("ADMIN_USERNAME")
	if expectedUsername == "" {
		expectedUsername = "admin" // Fallback for demonstration
		log.Println("Warning: ADMIN_USERNAME environment variable not set. Using default 'admin'.")
	}

	expectedPassword := os.Getenv("ADMIN_PASSWORD")
	if expectedPassword == "" {
		expectedPassword = "password" // Fallback for demonstration
		log.Println("Warning: ADMIN_PASSWORD environment variable not set. Using default 'password'.")
	}

	return username == expectedUsername && password == expectedPassword
}

// Login sets the user session to logged in.
func (s *AuthService) Login(w http.ResponseWriter, r *http.Request) error {
	session, err := s.store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Values[sessionKey] = true
	return session.Save(r, w)
}

// Logout sets the user session to logged out.
func (s *AuthService) Logout(w http.ResponseWriter, r *http.Request) error {
	session, err := s.store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Values[sessionKey] = false
	session.Options.MaxAge = -1 // Expire the cookie immediately
	return session.Save(r, w)
}

// IsLoggedIn checks if the user is currently logged in.
func (s *AuthService) IsLoggedIn(r *http.Request) bool {
	session, err := s.store.Get(r, sessionName)
	if err != nil {
		return false
	}
	loggedIn, ok := session.Values[sessionKey].(bool)
	return ok && loggedIn
}

// Middleware provides an authentication middleware.
func (s *AuthService) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.IsLoggedIn(r) {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}
