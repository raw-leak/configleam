package interfaces

import (
	"context"

	"github.com/raw-leak/configleam/internal/app/configleam-access/repository"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
)

type Keys interface {
	GenerateKey(ctx context.Context) (string, error)
}

type Repository interface {
	StoreAccessKey(ctx context.Context, accessKey repository.AccessKey) error
	RemoveKeys(ctx context.Context, keys []string) error
	GetAccessKeyPermissions(ctx context.Context, key string) (*permissions.AccessKeyPermissions, bool, error)
	PaginateAccessKeys(ctx context.Context, page int, size int) (*repository.PaginatedAccessKeys, error)
}

type Permissions interface {
	NewAccessKeyPermissions() *permissions.AccessKeyPermissions
	GetAvailableAccessKeyPermissions() []permissions.SinglePermission
}
