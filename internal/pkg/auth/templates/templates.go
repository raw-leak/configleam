package templates

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	LoginTemplate      = "login.html"
	LoginErrorTemplate = "login-error.html"
)

type AuthTemplates struct {
	tmpl *template.Template
}

func New() *AuthTemplates {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// this works when running the app
	templatesPath := filepath.Join(dir, "internal/pkg/auth/templates/*.html")

	// this works with tests
	// templatesPath := filepath.Join(dir, "./templates/*.html")

	tmpl := template.Must(template.ParseGlob(templatesPath))

	return &AuthTemplates{tmpl: tmpl}
}

func (t AuthTemplates) Login(w http.ResponseWriter, errMsg string) {
	err := t.tmpl.ExecuteTemplate(w, LoginTemplate, map[string]string{"ErrorMessage": errMsg})
	if err != nil {
		log.Printf("Error generating '%s': %v", LoginTemplate, err)
		http.Error(w, "Error rendering login section", http.StatusInternalServerError)
		return
	}

}

func (t AuthTemplates) LoginError(w http.ResponseWriter, errMsg string) {
	err := t.tmpl.ExecuteTemplate(w, LoginErrorTemplate, map[string]string{"ErrorMessage": errMsg})
	if err != nil {
		log.Printf("Error generating '%s': %v", LoginErrorTemplate, err)
		http.Error(w, "Error rendering login section", http.StatusInternalServerError)
		return
	}
}
