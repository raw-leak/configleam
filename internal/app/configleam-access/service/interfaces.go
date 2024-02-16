package service

import (
	"context"

	"github.com/raw-leak/configleam/internal/pkg/permissions"
)

type Keys interface {
	GenerateKey(ctx context.Context) (string, error)
}

type Repository interface {
	StoreKeyWithPermissions(ctx context.Context, key string, perms permissions.AccessKeyPermissions, meta map[string]string) error
	RemoveKeys(ctx context.Context, keys []string) error
	GetKeyPermissions(ctx context.Context, key string) (*permissions.AccessKeyPermissions, bool, error)
}

type Permissions interface {
	NewAccessKeyPermissions() *permissions.AccessKeyPermissions
}
