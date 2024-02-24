package service

import (
	"context"
	"time"

	dtoAccess "github.com/raw-leak/configleam/internal/app/access/dto"
	"github.com/raw-leak/configleam/internal/app/access/repository"
	"github.com/raw-leak/configleam/internal/app/dashboard/dto"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
)

type ConfigurationService interface {
	GetEnvs(ctx context.Context) []string
}

type AccessService interface {
	GenerateAccessKey(ctx context.Context, accessKeyPerms dtoAccess.AccessKeyPermissionsDto) (dtoAccess.AccessKeyPermissionsDto, error)
	DeleteAccessKeys(ctx context.Context, keys []string) error
	PaginateAccessKeys(ctx context.Context, page, size int) (*repository.PaginatedAccessKeys, error)
	GetAvailableAccessKeyPermissions(ctx context.Context) []permissions.SinglePermission
}

type Dashboard struct {
	accessService        AccessService
	configurationService ConfigurationService
}

// New creates a new instance of ConfigleamDashboard service.
func New(accessService AccessService, configurationService ConfigurationService) *Dashboard {
	return &Dashboard{
		accessService:        accessService,
		configurationService: configurationService,
	}
}

func (a *Dashboard) DashboardAccessKeys(ctx context.Context, page, size int) (dto.AccessParams, error) {
	paginated, err := a.accessService.PaginateAccessKeys(ctx, page, size)
	if err != nil {
		return dto.AccessParams{}, err
	}

	items := []map[string]string{}

	for _, item := range paginated.Items {
		var expiration string

		if item.ExpirationDate.IsZero() {
			expiration = "-"
		} else if item.ExpirationDate.After(time.Now()) {
			expiration = item.ExpirationDate.Format("2006-01-02T15:04:05Z07:00")
		} else {
			expiration = "Expired"
		}

		mappedItem := map[string]string{
			"Name":           item.Name,
			"MaskedKey":      item.MaskedKey,
			"CreationDate":   item.CreationDate.Format("2006-01-02T15:04:05Z07:00"),
			"ExpirationDate": expiration,
			"Key":            item.Key,
		}

		items = append(items, mappedItem)
	}

	paginationPages := []int{}
	for i := 1; i <= paginated.Pages; i++ {
		paginationPages = append(paginationPages, i)
	}

	ap := dto.AccessParams{
		Page:            paginated.Page,
		Pages:           paginated.Pages,
		Items:           items,
		Size:            paginated.Size,
		Total:           paginated.Total,
		PaginationPages: paginationPages,
	}

	return ap, nil
}

func (a *Dashboard) GetConfigEnvs(ctx context.Context) []string {
	return a.configurationService.GetEnvs(ctx)
}

func (a *Dashboard) CreateAccessKeyParams(ctx context.Context) dto.CreateAccessKeyParams {
	perms := a.accessService.GetAvailableAccessKeyPermissions(ctx)

	permsMap := []map[string]string{}

	for _, perm := range perms {
		permMap := map[string]string{"Label": perm.Label, "Tooltip": perm.Tooltip, "Value": perm.Value}
		permsMap = append(permsMap, permMap)
	}

	envs := a.configurationService.GetEnvs(ctx)

	return dto.CreateAccessKeyParams{Perms: permsMap, Envs: envs}
}

func (a Dashboard) CreateAccessKey(ctx context.Context, accessKeyPerms dtoAccess.AccessKeyPermissionsDto) (dto.CreatedAccessKey, error) {
	res, err := a.accessService.GenerateAccessKey(ctx, accessKeyPerms)
	if err != nil {
		return dto.CreatedAccessKey{}, err
	}

	return dto.CreatedAccessKey{AccessKey: res.AccessKey}, nil
}

func (a Dashboard) DeleteAccessKey(ctx context.Context, key string) error {
	err := a.accessService.DeleteAccessKeys(ctx, []string{key})
	if err != nil {
		return err
	}
	return nil
}
