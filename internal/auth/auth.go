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

// AuthService provides authentication services.
type AuthService struct {
	store sessions.Store
}

var OsExit = os.Exit
var FatalLogger = log.Fatal

// NewAuthService creates a new AuthService with the given session key.
func NewAuthService(sessionKey string) *AuthService {
	if sessionKey == "" {
		FatalLogger("Session key not provided")
	}
	store := sessions.NewCookieStore([]byte(sessionKey))
	return &AuthService{store: store}
}

// Authenticate checks user credentials.
// In a real application, this would check against a database.
func (s *AuthService) Authenticate(username, password string) bool {
	expectedUsername := os.Getenv("ADMIN_USERNAME")
	// No fallback. If not set, authentication will fail.

	expectedPassword := os.Getenv("ADMIN_PASSWORD")
	// No fallback. If not set, authentication will fail.

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
