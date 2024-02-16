package httpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raw-leak/configleam/internal/pkg/permissions"
	"github.com/raw-leak/configleam/internal/pkg/transport/httpserver"
	"github.com/stretchr/testify/suite"
)

type AuthMiddlewareTestSuite struct {
	suite.Suite
	access     *MockConfigleamAccessService
	perms      httpserver.PermissionsBuilder
	authMiddle *httpserver.AuthMiddleware
}

type MockConfigleamAccessService struct {
	mockPermCheck func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error)
}

func TestAuthMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(AuthMiddlewareTestSuite))
}

func (m *MockConfigleamAccessService) GetAccessKeyPermissions(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
	return m.mockPermCheck(ctx, accessKey)
}

func (suite *AuthMiddlewareTestSuite) SetupTest() {
	suite.access = &MockConfigleamAccessService{}
	suite.perms = permissions.New()
	suite.authMiddle = httpserver.NewAuthMiddleware(suite.access, suite.perms)
}

func (suite *AuthMiddlewareTestSuite) TestAuthMiddleware() {
	testCases := []struct {
		name               string
		accessKey          string
		requiredPermission permissions.Operation
		prepareMock        func()
		expectedStatus     int
	}{
		// admin access key
		{
			name:               "Access granted with admin key and required ReadConfig",
			accessKey:          "admin-key",
			requiredPermission: permissions.ReadConfig,
			prepareMock: func() {
				suite.access.mockPermCheck = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
					perms := suite.perms.NewAccessKeyPermissions()
					perms.SetAdmin()
					return perms, true, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:               "Access granted with admin key and required clone-env",
			accessKey:          "admin-key",
			requiredPermission: permissions.CloneEnvironment,
			prepareMock: func() {
				suite.access.mockPermCheck = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
					perms := suite.perms.NewAccessKeyPermissions()
					perms.SetAdmin()
					return perms, true, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:               "Access granted with admin key and required dashboard access",
			accessKey:          "admin-key",
			requiredPermission: permissions.AccessDashboard,
			prepareMock: func() {
				suite.access.mockPermCheck = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
					perms := suite.perms.NewAccessKeyPermissions()
					perms.SetAdmin()
					return perms, true, nil

				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:               "Access granted with admin key and required multiple permissions",
			accessKey:          "admin-key",
			requiredPermission: permissions.AccessDashboard | permissions.ReadConfig | permissions.RevealSecrets | permissions.CloneEnvironment,
			prepareMock: func() {
				suite.access.mockPermCheck = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
					perms := suite.perms.NewAccessKeyPermissions()
					perms.SetAdmin()
					return perms, true, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		// Insufficient permissions
		{
			name:               "Access denied with Insufficient permissions key for cloning env for env",
			accessKey:          "user-key",
			requiredPermission: permissions.CloneEnvironment,
			prepareMock: func() {
				suite.access.mockPermCheck = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
					perms := suite.perms.NewAccessKeyPermissions()
					perms.Grant("default", permissions.ReadConfig) // Only read permission
					return perms, true, nil

				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:               "Access denied with Insufficient permissions key for reading config for env",
			accessKey:          "user-key",
			requiredPermission: permissions.ReadConfig,
			prepareMock: func() {
				suite.access.mockPermCheck = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
					perms := suite.perms.NewAccessKeyPermissions()
					perms.Grant("default", permissions.CloneEnvironment) // Only read permission
					return perms, true, nil

				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:               "Access denied with Insufficient permissions key for accessing dashboard",
			accessKey:          "access-key",
			requiredPermission: permissions.AccessDashboard,
			prepareMock: func() {
				suite.access.mockPermCheck = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
					perms := suite.perms.NewAccessKeyPermissions()
					perms.Grant("default", permissions.CloneEnvironment)
					return perms, true, nil

				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:      "Access denied with Insufficient permissions key for accessing dashboard",
			accessKey: "access-key",
			prepareMock: func() {
				suite.access.mockPermCheck = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
					perms := suite.perms.NewAccessKeyPermissions()
					perms.Grant("default", permissions.CloneEnvironment)
					return perms, true, nil

				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.prepareMock()

			req, _ := http.NewRequest("GET", "/", nil)
			req.Header.Set("X-Access-Key", tc.accessKey)

			rr := httptest.NewRecorder()

			handler := suite.authMiddle.Guard(tc.requiredPermission)(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK) // Dummy handler that should only be called if permissions check passes
			})

			handler.ServeHTTP(rr, req)

			suite.Equal(tc.expectedStatus, rr.Code)
		})
	}
}
