package httpserver

import (
	"context"
	"net/http"

	"github.com/raw-leak/configleam/internal/app/configleam-access/dto"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
)

type ConfigleamSet interface {
	ConfigleamService
	ConfigleamEndpoints
}

type ConfigleamService interface {
	HealthCheck(ctx context.Context) error
}

type ConfigleamEndpoints interface {
	CloneConfigHandler(w http.ResponseWriter, r *http.Request)
	ReadConfigHandler(w http.ResponseWriter, r *http.Request)
	DeleteConfigHandler(w http.ResponseWriter, r *http.Request)
}

type ConfigleamSecretsSet interface {
	ConfigleamSecretsService
	ConfigleamSecretsEndpoints
}

type ConfigleamSecretsEndpoints interface {
	UpsertSecretsHandler(w http.ResponseWriter, r *http.Request)
}

type ConfigleamSecretsService interface {
	UpsertSecrets(ctx context.Context, env string, secrets map[string]interface{}) error
}

type ConfigleamAccessSet interface {
	ConfigleamAccessService
	ConfigleamAccessEndpoints
}
type ConfigleamAccessEndpoints interface {
	GenerateAccessKeyHandler(w http.ResponseWriter, r *http.Request)
	DeleteAccessKeysHandler(w http.ResponseWriter, r *http.Request)
}

type ConfigleamAccessService interface {
	GetAccessKeyPermissions(ctx context.Context, key string) (*permissions.AccessKeyPermissions, bool, error)
	GenerateAccessKey(ctx context.Context, accessKeyPerms dto.AccessKeyPermissionsDto) (dto.AccessKeyPermissionsDto, error)
	DeleteAccessKeys(ctx context.Context, keys []string) error
}

type PermissionsBuilder interface {
	NewAccessKeyPermissions() *permissions.AccessKeyPermissions
}

type ConfigleamDashboardEndpoints interface {
	RenderLoginWithErrorPageHandler(w http.ResponseWriter, r *http.Request, errMsg string)
	RenderErrorPageHandler(w http.ResponseWriter, r *http.Request, errMsg string)
}
