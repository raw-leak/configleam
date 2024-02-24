package httpserver

import (
	"context"
	"net/http"

	"github.com/raw-leak/configleam/internal/app/access/dto"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
)

// configuration
type ConfigurationSet interface {
	ConfigurationService
	ConfigurationEndpoints
}

type ConfigurationService interface {
	HealthCheck(ctx context.Context) error
}

type ConfigurationEndpoints interface {
	CloneConfigHandler(w http.ResponseWriter, r *http.Request)
	ReadConfigHandler(w http.ResponseWriter, r *http.Request)
	DeleteConfigHandler(w http.ResponseWriter, r *http.Request)
}

// secrets
type SecretsSet interface {
	SecretsService
	SecretsEndpoints
}

type SecretsEndpoints interface {
	UpsertSecretsHandler(w http.ResponseWriter, r *http.Request)
}

type SecretsService interface {
	UpsertSecrets(ctx context.Context, env string, secrets map[string]interface{}) error
}

// access
type AccessSet interface {
	AccessService
	AccessEndpoints
}
type AccessEndpoints interface {
	GenerateAccessKeyHandler(w http.ResponseWriter, r *http.Request)
	DeleteAccessKeysHandler(w http.ResponseWriter, r *http.Request)
}

type AccessService interface {
	GetAccessKeyPermissions(ctx context.Context, key string) (*permissions.AccessKeyPermissions, bool, error)
	GenerateAccessKey(ctx context.Context, accessKeyPerms dto.AccessKeyPermissionsDto) (dto.AccessKeyPermissionsDto, error)
	DeleteAccessKeys(ctx context.Context, keys []string) error
}

// dashboard
type DashboardSet interface {
	DashboardEndpoints
}

type DashboardEndpoints interface {
	HomeHandler(w http.ResponseWriter, r *http.Request)
	ConfigHandler(w http.ResponseWriter, r *http.Request)
	AccessHandler(w http.ResponseWriter, r *http.Request)
	CreateAccessKeyParamsHandler(w http.ResponseWriter, r *http.Request)
	CreateAccessKeyHandler(w http.ResponseWriter, r *http.Request)
	DeleteAccessKeyHandler(w http.ResponseWriter, r *http.Request)
}

// other
type PermissionsBuilder interface {
	NewAccessKeyPermissions() *permissions.AccessKeyPermissions
}
