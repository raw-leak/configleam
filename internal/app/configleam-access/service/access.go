package service

import (
	"context"

	"github.com/raw-leak/configleam/internal/app/configleam-access/dto"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
)

type ConfigleamAccess struct {
	keys       Keys
	perms      Permissions
	repository Repository
}

// New creates a new instance of ConfigleamAccess service.
func New(keys Keys, perms Permissions, repository Repository) *ConfigleamAccess {
	return &ConfigleamAccess{
		keys:       keys,
		perms:      perms,
		repository: repository,
	}
}

type EnvPerms struct {
	Environment string
	Operations  permissions.Operation
}

func (a *ConfigleamAccess) GenerateAccessKey(ctx context.Context, accessKeyPerms dto.AccessKeyPermissionsDto) (dto.AccessKeyPermissionsDto, error) {
	key, err := a.keys.GenerateKey(ctx)
	if err != nil {
		return newEmptyAccessKeyPermissionsDto(), err
	}

	err = a.repository.StoreKeyWithPermissions(ctx, key, accessKeyPerms.ToAccessKeyPermissions(), accessKeyPerms.ToMeta())
	if err != nil {
		return newEmptyAccessKeyPermissionsDto(), err
	}

	accessKeyPerms.AccessKey = key

	return accessKeyPerms, nil
}

func (a *ConfigleamAccess) GetAccessKeyPermissions(ctx context.Context, key string) (*permissions.AccessKeyPermissions, bool, error) {
	perms, ok, err := a.repository.GetKeyPermissions(ctx, key)
	if err != nil {
		return nil, false, err
	}

	return perms, ok, nil
}

func (a *ConfigleamAccess) DeleteAccessKeys(ctx context.Context, keys []string) error {
	err := a.repository.RemoveKeys(ctx, keys)
	if err != nil {
		return err
	}

	return nil
}

func newEmptyAccessKeyPermissionsDto() dto.AccessKeyPermissionsDto {
	return dto.AccessKeyPermissionsDto{
		Envs: make(map[string]dto.EnvironmentPermissions),
	}
}
