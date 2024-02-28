package service

import (
	"context"
	"time"

	"github.com/raw-leak/configleam/internal/app/access/dto"
	"github.com/raw-leak/configleam/internal/app/access/interfaces"
	"github.com/raw-leak/configleam/internal/app/access/repository"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
)

type AccessService struct {
	keys       interfaces.Keys
	perms      interfaces.Permissions
	repository interfaces.AccessRepository
}

// TODO: test
// New creates a new instance of AccessService service.
func New(keys interfaces.Keys, perms interfaces.Permissions, repository interfaces.AccessRepository) *AccessService {
	return &AccessService{
		keys:       keys,
		perms:      perms,
		repository: repository,
	}
}

type EnvPerms struct {
	Environment string
	Operations  permissions.Operation
}

func (a *AccessService) GenerateAccessKey(ctx context.Context, accessKeyPerms dto.AccessKeyPermissionsDto) (dto.AccessKeyPermissionsDto, error) {
	key, err := a.keys.GenerateKey(ctx)
	if err != nil {
		return newEmptyAccessKeyPermissionsDto(), err
	}

	perms := accessKeyPerms.ToAccessKeyPermissions()
	accessKey := repository.AccessKey{
		Key:   key,
		Perms: perms,
		Metadata: repository.AccessKeyMetadata{
			Name:           accessKeyPerms.Name,
			MaskedKey:      a.GetMaskedKey(key),
			ExpirationDate: accessKeyPerms.ExpDate,
			CreationDate:   time.Now(),
			Permissions:    perms,
		},
	}

	err = a.repository.StoreAccessKey(ctx, accessKey)
	if err != nil {
		return newEmptyAccessKeyPermissionsDto(), err
	}

	accessKeyPerms.AccessKey = key // it will be shown only once

	return accessKeyPerms, nil
}

func (a *AccessService) GetAccessKeyPermissions(ctx context.Context, key string) (*permissions.AccessKeyPermissions, bool, error) {
	perms, ok, err := a.repository.GetAccessKeyPermissions(ctx, key)
	if err != nil {
		return nil, false, err
	}

	return perms, ok, nil
}

func (a *AccessService) PaginateAccessKeys(ctx context.Context, page, size int) (*repository.PaginatedAccessKeys, error) {
	paginated, err := a.repository.PaginateAccessKeys(ctx, page, size)
	if err != nil {
		return nil, err
	}

	return paginated, nil
}

func (a *AccessService) DeleteAccessKeys(ctx context.Context, keys []string) error {
	err := a.repository.RemoveKeys(ctx, keys)
	if err != nil {
		return err
	}

	return nil
}

func (a *AccessService) GetMaskedKey(key string) string {
	return key[:8] + "****" + key[len(key)-4:]
}

func (a *AccessService) GetAvailableAccessKeyPermissions(_ context.Context) []permissions.SinglePermission {
	return a.perms.GetAvailableAccessKeyPermissions()
}

func newEmptyAccessKeyPermissionsDto() dto.AccessKeyPermissionsDto {
	return dto.AccessKeyPermissionsDto{
		Envs: make(map[string]dto.EnvironmentPermissions),
	}
}
