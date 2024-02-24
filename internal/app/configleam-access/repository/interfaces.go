package repository

import (
	"time"

	"github.com/raw-leak/configleam/internal/pkg/permissions"
)

type AccessKeyMetadata struct {
	Key            string
	Name           string                           `json:"name"`
	MaskedKey      string                           `json:"masked_key"`
	ExpirationDate time.Time                        `json:"expiration_date"`
	CreationDate   time.Time                        `json:"creation_date"`
	Permissions    permissions.AccessKeyPermissions `json:"permissions"`
}

type AccessKey struct {
	Key      string
	Perms    permissions.AccessKeyPermissions
	Metadata AccessKeyMetadata
}

type PaginatedAccessKeys struct {
	Total int                 `json:"total"`
	Pages int                 `json:"pages"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
	Items []AccessKeyMetadata `json:"items"`
}
