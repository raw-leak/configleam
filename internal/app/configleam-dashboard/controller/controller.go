package controller

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	dtoAccess "github.com/raw-leak/configleam/internal/app/configleam-access/dto"
	"github.com/raw-leak/configleam/internal/app/configleam-dashboard/dto"
	"github.com/raw-leak/configleam/internal/app/configleam-dashboard/templates"
)

var tmpl *template.Template

type Service interface {
	DashboardAccessKeys(context.Context, int, int) (dto.AccessParams, error)
	CreateAccessKeyParams(context.Context) dto.CreateAccessKeyParams
	CreateAccessKey(context.Context, dtoAccess.AccessKeyPermissionsDto) (dto.CreatedAccessKey, error)
	DeleteAccessKey(context.Context, string) error
}

type ConfigleamDashboardEndpoints struct {
	service   Service
	templates *templates.DashboardTemplates
}

func New(service Service, templates *templates.DashboardTemplates) *ConfigleamDashboardEndpoints {
	return &ConfigleamDashboardEndpoints{service: service, templates: templates}
}

func (e ConfigleamDashboardEndpoints) HomeHandler(w http.ResponseWriter, r *http.Request) {
	err := e.templates.Home(w)
	if err != nil {
		e.templates.ErrorSection(w, err.Error())
	}
}

func (e ConfigleamDashboardEndpoints) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	err := e.templates.Config(w, nil)
	if err != nil {
		e.templates.ErrorSection(w, err.Error())
	}
}

func (e ConfigleamDashboardEndpoints) AccessHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	pageStr := query.Get("page")
	sizeStr := query.Get("size")

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		http.Error(w, "Page must be a valid number", http.StatusBadRequest)
		return
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		http.Error(w, "Size must be a valid number", http.StatusBadRequest)
		return
	}

	if page == 0 {
		page = 1
	}

	if size == 0 {
		size = 10
	}

	payload, err := e.service.DashboardAccessKeys(r.Context(), page, size)
	if err != nil {
		log.Printf("Error loading dashboard access data: %v", err)
		e.templates.ErrorSection(w, err.Error())
		return
	}

	err = e.templates.Access(w, payload)
	if err != nil {
		e.templates.ErrorSection(w, err.Error())
	}
}

func (e ConfigleamDashboardEndpoints) CreateAccessKeyParamsHandler(w http.ResponseWriter, r *http.Request) {
	params := e.service.CreateAccessKeyParams(r.Context())
	err := e.templates.CreateAccessKeyParams(w, params)
	if err != nil {
		e.templates.ErrorSection(w, err.Error())
	}
}

func (e ConfigleamDashboardEndpoints) CreateAccessKeyHandler(w http.ResponseWriter, r *http.Request) {
	dto, err := parseCreateAccessKeyForm(r)
	if err != nil {
		log.Printf("Error parsing access keys params: %v", err)
		e.templates.ErrorSection(w, err.Error())
		return
	}

	createdAccessKey, err := e.service.CreateAccessKey(r.Context(), *dto)
	if err != nil {
		log.Printf("Error creating access keys through dashboard error: %v", err)
		e.templates.ErrorSection(w, err.Error())
		return
	}

	err = e.templates.CreatedAccessKey(w, createdAccessKey)
	if err != nil {
		e.templates.ErrorSection(w, err.Error())
		return
	}
}

func (e ConfigleamDashboardEndpoints) DeleteAccessKeyHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	accessKey := r.FormValue("accessKey")
	if len(accessKey) < 1 {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := e.service.DeleteAccessKey(r.Context(), accessKey)
	if err != nil {
		log.Printf("Error creating access keys through dashboard error: %v", err)
		e.templates.ErrorSection(w, err.Error())
		return
	}

	err = e.templates.DeletedAccessKey(w)
	if err != nil {
		e.templates.ErrorSection(w, err.Error())
		return
	}
}

func parseCreateAccessKeyForm(r *http.Request) (*dtoAccess.AccessKeyPermissionsDto, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	accessKeyParams := &dtoAccess.AccessKeyPermissionsDto{
		Envs: make(map[string]dtoAccess.EnvironmentPermissions),
	}

	for key, values := range r.Form {
		// Split the key to identify environment and permission
		parts := strings.Split(key, "[")
		if len(parts) == 1 { // Non-environment fields
			switch key {
			case "access-key-name":
				accessKeyParams.Name = values[0]
			case "expiration-date":
				// Parse the date accordingly
				accessKeyParams.ExpDate, _ = time.Parse("2006-01-02", values[0])
			}
			continue
		}

		// Environment specific fields
		envName := strings.Trim(parts[1], "[]")
		permName := strings.Trim(parts[2], "[]:")
		env, exists := accessKeyParams.Envs[envName]
		if !exists {
			env = dtoAccess.EnvironmentPermissions{}
		}

		value := values[0] == "true"

		// Reflect can be used here for a more dynamic approach, or a switch
		switch permName {
		case "envAdminAccess":
			env.EnvAdminAccess = value
		case "readConfig":
			env.ReadConfig = value
		case "revealSecrets":
			env.RevealSecrets = value
		case "cloneEnvironment":
			env.CloneEnvironment = value
		case "createSecrets":
			env.CreateSecrets = value
		case "accessDashboard":
			env.AccessDashboard = value
		}

		accessKeyParams.Envs[envName] = env
	}

	return accessKeyParams, nil
}
