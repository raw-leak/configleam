package templates

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/raw-leak/configleam/internal/app/dashboard/dto"
)

const (
	HomeTemplate             = "home.html"
	ConfigTemplate           = "config.html"
	AccessTemplate           = "access.html"
	ErrorTemplate            = "error.html"
	CreateAccessKeyTemplate  = "create-access-key.html"
	CreatedAccessKeyTemplate = "created-access-key.html"
	DeletedAccessKeyTemplate = "deleted-access-key.html"
)

var funcMap = template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},
	"sub": func(a, b int) int {
		return a - b
	},
	"seq": func(start, end int) []int {
		var seq []int
		for i := start; i <= end; i++ {
			seq = append(seq, i)
		}
		return seq
	},
}

type DashboardTemplates struct {
	tmpl *template.Template
}

func New() *DashboardTemplates {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	templatesPath := filepath.Join(dir, "internal/app/dashboard/templates/*.html")

	tmpl := template.New("").Funcs(funcMap)
	tmpl, err = tmpl.ParseGlob(templatesPath)
	tmpl = template.Must(tmpl, err)

	return &DashboardTemplates{tmpl: tmpl}
}

// home
func (t DashboardTemplates) Home(w http.ResponseWriter) error {
	err := t.tmpl.ExecuteTemplate(w, HomeTemplate, nil)
	if err != nil {
		log.Printf("Error generating '%s' in dashboard: %v", HomeTemplate, err)
	}

	return err
}

// access
func (t DashboardTemplates) Access(w http.ResponseWriter, payload dto.AccessParams) error {
	err := t.tmpl.ExecuteTemplate(w, AccessTemplate, payload)
	if err != nil {
		log.Printf("Error generating '%s' in dashboard: %v", AccessTemplate, err)
	}

	return err
}

func (t DashboardTemplates) CreatedAccessKey(w http.ResponseWriter, payload dto.CreatedAccessKey) error {
	err := t.tmpl.ExecuteTemplate(w, CreatedAccessKeyTemplate, map[string]string{
		"AccessKey": payload.AccessKey,
	})
	if err != nil {
		log.Printf("Error generating '%s' in dashboard: %v", CreatedAccessKeyTemplate, err)
	}
	return err
}

func (t DashboardTemplates) DeletedAccessKey(w http.ResponseWriter) error {
	err := t.tmpl.ExecuteTemplate(w, DeletedAccessKeyTemplate, nil)
	if err != nil {
		log.Printf("Error generating '%s' in dashboard: %v", DeletedAccessKeyTemplate, err)
	}
	return err
}

func (t DashboardTemplates) CreateAccessKeyParams(w http.ResponseWriter, payload dto.CreateAccessKeyParams) error {
	err := t.tmpl.ExecuteTemplate(w, CreateAccessKeyTemplate, map[string]any{
		"Envs":  payload.Envs,
		"Perms": payload.Perms,
	})
	if err != nil {
		log.Printf("Error generating '%s' in dashboard: %v", CreateAccessKeyTemplate, err)
	}
	return err
}

// config
func (t DashboardTemplates) Config(w http.ResponseWriter, payload map[string]any) error {
	err := t.tmpl.ExecuteTemplate(w, ConfigTemplate, payload)
	if err != nil {
		log.Printf("Error generating '%s' in dashboard: %v", ConfigTemplate, err)
	}

	return err
}

// error
func (t DashboardTemplates) ErrorSection(w http.ResponseWriter, errMsg string) {
	err := t.tmpl.ExecuteTemplate(w, ErrorTemplate, map[string]any{
		"Message": errMsg,
	})
	if err != nil {
		log.Printf("Error generating '%s' in dashboard error: %v", ErrorTemplate, err)
		http.Error(w, "Error rendering error section", http.StatusInternalServerError)
		return
	}
}
