package auth

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/raw-leak/configleam/internal/app/access/dto"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
)

const (
	AdminUsernameEnv = "CG_ADMIN_USERNAME"
	AdminPasswordEnv = "CG_ADMIN_PASSWORD"
	AccessKeyHeader  = "X-Access-Key"
	AccessKeyCookies = "AccessKey"
)

type AccessKeyContextKey struct{}

type AccessService interface {
	GetAccessKeyPermissions(ctx context.Context, key string) (*permissions.AccessKeyPermissions, bool, error)
	GenerateAccessKey(ctx context.Context, accessKeyPerms dto.AccessKeyPermissionsDto) (dto.AccessKeyPermissionsDto, error)
	DeleteAccessKeys(ctx context.Context, keys []string) error
}

type Templates interface {
	Login(w http.ResponseWriter, errMsg string)
	LoginError(w http.ResponseWriter, errMsg string)
}

type PermissionsBuilder interface {
	NewAccessKeyPermissions() *permissions.AccessKeyPermissions
}

// AuthMiddleware holds the service needed to validate permissions
type AuthMiddleware struct {
	access    AccessService
	perms     PermissionsBuilder
	templates Templates
}

// NewAuthMiddleware creates a new instance of AuthMiddleware
func NewAuthMiddleware(access AccessService, perms PermissionsBuilder, templates Templates) *AuthMiddleware {
	return &AuthMiddleware{
		access:    access,
		perms:     perms,
		templates: templates,
	}
}

// Guard creates a middleware that checks for the required permissions
func (m *AuthMiddleware) Guard(requiredPermission permissions.Operation) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			accessKey, query := r.Header.Get(AccessKeyHeader), r.URL.Query()

			if accessKey == "" {
				http.Error(w, "Access key required", http.StatusUnauthorized)
				return
			}

			accessKeyPerms, ok, err := m.access.GetAccessKeyPermissions(r.Context(), accessKey)
			if err != nil {
				http.Error(w, "Error checking permissions", http.StatusInternalServerError)
				return
			}
			if !ok {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			hasPermission := accessKeyPerms.Can(query.Get("env"), requiredPermission)
			if !hasPermission {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			ctxWithPermissions := context.WithValue(r.Context(), AccessKeyContextKey{}, *accessKeyPerms)
			r = r.WithContext(ctxWithPermissions)

			next.ServeHTTP(w, r)
		}
	}
}

// GuardDashboard creates a middleware that checks for the required permissions for dashboard
func (m *AuthMiddleware) GuardDashboard() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AccessKeyCookies)
			if err != nil && err != http.ErrNoCookie {
				http.Redirect(w, r, "dashboard/login", http.StatusSeeOther)
				return
			}

			if err == http.ErrNoCookie {
				http.Redirect(w, r, "dashboard/login", http.StatusSeeOther)
				return
			}

			accessKeyPerms, ok, err := m.access.GetAccessKeyPermissions(r.Context(), cookie.Value)
			if err != nil {
				http.Redirect(w, r, "dashboard/login", http.StatusSeeOther)
				return
			}
			if !ok {
				http.Redirect(w, r, "dashboard/login", http.StatusSeeOther)
				return
			}

			hasPermission := accessKeyPerms.IsGlobalAdmin()
			if !hasPermission {
				http.Redirect(w, r, "dashboard/login", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		}
	}
}

func (m *AuthMiddleware) LoginHandler(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie(AccessKeyCookies)
	if err != nil && err != http.ErrNoCookie {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err == http.ErrNoCookie {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		username, password := r.FormValue("username"), r.FormValue("password")
		if len(username) < 1 && len(password) < 1 {
			m.templates.Login(w, "")
			return
		}

		if len(username) < 1 || len(password) < 1 {
			m.templates.Login(w, "Access to dashboard forbidden")
			return
		}

		if len(os.Getenv(AdminUsernameEnv)) < 1 && len(os.Getenv(AdminPasswordEnv)) < 1 {
			m.templates.Login(w, "Access to dashboard forbidden")
			return
		}

		if username != os.Getenv(AdminUsernameEnv) || password != os.Getenv(AdminPasswordEnv) {
			m.templates.Login(w, "Access to dashboard forbidden")
			return
		}

		expiresIn := time.Now().Add(30 * time.Minute)
		accessKey, err := m.access.GenerateAccessKey(r.Context(),
			dto.AccessKeyPermissionsDto{
				GlobalAdmin: true,
				Name:        "Temporal dashboard access",
				ExpDate:     expiresIn,
			},
		)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:  AccessKeyCookies,
			Value: accessKey.AccessKey,
			Path:  "dashboard",
			// HttpOnly: true,
			// Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Expires:  expiresIn,
		})

	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (m AuthMiddleware) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(AccessKeyCookies)
	if err != nil && err != http.ErrNoCookie {
		http.Redirect(w, r, "dashboard/login", http.StatusSeeOther)
		return
	}

	if err == http.ErrNoCookie {
		http.Redirect(w, r, "dashboard/login", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    AccessKeyCookies,
		Value:   cookie.Value,
		Path:    "dashboard",
		Expires: time.Unix(0, 0),
		MaxAge:  -1,
	})

	err = m.access.DeleteAccessKeys(r.Context(), []string{cookie.Value})
	if err != nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
