package auth

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
)

const (
	sessionName = "gemini-session"
	sessionKey  = "user_logged_in"
)

// AuthServiceInterface defines the interface for authentication services.
type AuthServiceInterface interface {
	Authenticate(username, password string) bool
	Login(w http.ResponseWriter, r *http.Request) error
	Logout(w http.ResponseWriter, r *http.Request) error
	IsLoggedIn(r *http.Request) bool
	Middleware(next http.Handler) http.Handler
}

// AuthService provides authentication services.
type AuthService struct {
	Store sessions.Store
}

var OsExit = os.Exit
var FatalLogger = log.Fatal

// NewAuthService creates a new AuthService with the given session key.
func NewAuthService(sessionKey string) AuthServiceInterface {
	if sessionKey == "" {
		FatalLogger("Session key not provided")
	}
	store := sessions.NewCookieStore([]byte(sessionKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	return &AuthService{Store: store}
}

// NewAuthServiceWithStore creates a new AuthService with the given session store.
func NewAuthServiceWithStore(store sessions.Store) AuthServiceInterface {
	return &AuthService{Store: store}
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
	session, err := s.Store.Get(r, sessionName)
	if err != nil {
		log.Printf("AuthService.Login: Error getting session: %v", err)
		return err
	}
	session.Values[sessionKey] = true
	err = session.Save(r, w)
	if err != nil {
		log.Printf("AuthService.Login: Error saving session: %v", err)
	}
	return err
}

// Logout sets the user session to logged out.
func (s *AuthService) Logout(w http.ResponseWriter, r *http.Request) error {
	session, err := s.Store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Values[sessionKey] = false
	session.Options.MaxAge = -1 // Expire the cookie immediately
	return session.Save(r, w)
}

// IsLoggedIn checks if the user is currently logged in.
func (s *AuthService) IsLoggedIn(r *http.Request) bool {
	session, err := s.Store.Get(r, sessionName)
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
