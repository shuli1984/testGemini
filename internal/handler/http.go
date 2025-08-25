package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gemini-demo/internal/auth"
	"gemini-demo/internal/models"
	"gemini-demo/internal/util"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Handler struct {
	DB          *gorm.DB
	AuthService *auth.AuthService
}

func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	siteData, err := models.GetSiteData(h.DB) // Use the new GetSiteData
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error getting site data: %v", err)
		return
	}

	projectRoot := util.ProjectRoot("")
	var templateFiles []string
	err = filepath.Walk(filepath.Join(projectRoot, "templates"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			templateFiles = append(templateFiles, path)
		}
		return nil
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error walking templates directory: %v", err)
		return
	}

	tmpl, err := template.ParseFiles(templateFiles...)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error parsing template (HelloHandler): %v", err)
		return
	}

	if err := tmpl.ExecuteTemplate(w, getTemplateName(r), siteData); err != nil { // Pass siteData
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error executing template (HelloHandler): %v", err)
		return
	}
}

func (h *Handler) PageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageName := vars["name"]

	pageData, err := models.GetPageData(h.DB, pageName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Page not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	projectRoot := util.ProjectRoot("")
	var templateFiles []string
	err = filepath.Walk(filepath.Join(projectRoot, "templates"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			templateFiles = append(templateFiles, path)
		}
		return nil
	})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error walking templates directory: %v", err)
		return
	}

	tmpl, err := template.ParseFiles(templateFiles...)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error parsing template (PageHandler): %v", err)
		return
	}

	if err := tmpl.ExecuteTemplate(w, getTemplateName(r), pageData); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error executing template (PageHandler): %v", err)
		return
	}
}

func (h *Handler) AboutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "This is the about page.")
}

func (h *Handler) UpdatePageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pageName := vars["name"]

	var updatedPage models.Page
	err := json.NewDecoder(r.Body).Decode(&updatedPage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if updatedPage.Name != pageName {
		http.Error(w, "Page name in URL and body do not match", http.StatusBadRequest)
		return
	}

	existingPage, err := models.GetPageData(h.DB, pageName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Page not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	existingPage.Title = updatedPage.Title
	existingPage.Description = updatedPage.Description
	existingPage.Message = updatedPage.Message

	if err := existingPage.UpdatePage(h.DB); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(existingPage)
}

// LoginHandler handles admin login requests.
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var err error // Declare err at the top of the function
	if h.AuthService.IsLoggedIn(r) {
		http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		projectRoot := util.ProjectRoot("")
		var templateFiles []string
		err = filepath.Walk(filepath.Join(projectRoot, "templates"), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
				templateFiles = append(templateFiles, path)
			}
			return nil
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var tmpl *template.Template // Declare tmpl here
		tmpl, err = template.ParseFiles(templateFiles...) // Assign to existing err
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = tmpl.ExecuteTemplate(w, getTemplateName(r), nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err = json.NewDecoder(r.Body).Decode(&credentials)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if h.AuthService.Authenticate(credentials.Username, credentials.Password) {
		err := h.AuthService.Login(w, r)
		if err != nil {
			http.Error(w, "Failed to log in", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Login successful"})
	} else {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
	}
}

// DashboardHandler displays system information after login.
func (h *Handler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	data, err := models.GetDashboardData(h.DB)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error getting dashboard data: %v", err)
		return
	}

	projectRoot := util.ProjectRoot("")
	var templateFiles []string
	walkErr := filepath.Walk(filepath.Join(projectRoot, "templates"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			templateFiles = append(templateFiles, path)
		}
		return nil
	})
	if walkErr != nil { // Use walkErr here
		http.Error(w, walkErr.Error(), http.StatusInternalServerError) // Use walkErr here
		return
	}

	tmpl, err := template.ParseFiles(templateFiles...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, getTemplateName(r), data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// AdminRedirectHandler redirects /admin to /admin/dashboard
func (h *Handler) AdminRedirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
}

func getTemplateName(r *http.Request) string {
	path := r.URL.Path
	if path == "/" {
		return "index.html"
	}
	if strings.HasPrefix(path, "/page/") {
		return "page.html"
	}
	return strings.Replace(strings.TrimPrefix(path, "/"), "/", "_", 1) + ".html"
}