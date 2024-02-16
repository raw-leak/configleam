package httpserver

import (
	"context"
	"net/http"
	"os"

	"github.com/raw-leak/configleam/internal/pkg/permissions"
)

const (
	AccessKeyHeader   = "X-Access-Key"
	AdminAccessKeyEnv = "CG_ADMIN_ACCESS_KEY"
)

type AccessKeyContextKey struct{}

// AuthMiddleware holds the service needed to validate permissions
type AuthMiddleware struct {
	access ConfigleamAccessService
	perms  PermissionsBuilder
}

// NewAuthMiddleware creates a new instance of AuthMiddleware
func NewAuthMiddleware(access ConfigleamAccessService, perms PermissionsBuilder) *AuthMiddleware {
	return &AuthMiddleware{
		access: access,
		perms:  perms,
	}
}

// AuthMid creates a middleware that checks for the required permissions
func (m *AuthMiddleware) Guard(requiredPermission permissions.Operation) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			accessKey := r.Header.Get(AccessKeyHeader)
			if accessKey == "" {
				http.Error(w, "Access key required", http.StatusUnauthorized)
				return
			}

			if accessKey == os.Getenv(AdminAccessKeyEnv) {
				adminPerms := m.perms.NewAccessKeyPermissions()
				adminPerms.SetAdmin()

				ctxWithPermissions := context.WithValue(r.Context(), AccessKeyContextKey{}, adminPerms)
				r = r.WithContext(ctxWithPermissions)

				next.ServeHTTP(w, r)
				return
			}

			accessKeyPerms, ok, err := m.access.GetAccessKeyPermissions(r.Context(), accessKey)
			if err != nil {
				http.Error(w, "Error checking permissions", http.StatusInternalServerError)
				return
			}

			if !ok {
				http.Error(w, "Error checking permissions", http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			hasPermission := accessKeyPerms.Can(query.Get("env"), requiredPermission)
			if !hasPermission {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}

			ctxWithPermissions := context.WithValue(r.Context(), AccessKeyContextKey{}, &accessKeyPerms)
			r = r.WithContext(ctxWithPermissions)

			next.ServeHTTP(w, r)
		}
	}
}
