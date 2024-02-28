package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raw-leak/configleam/internal/app/access/dto"
	"github.com/raw-leak/configleam/internal/pkg/auth"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
	"github.com/raw-leak/configleam/internal/pkg/transport/httpserver"
	"github.com/stretchr/testify/suite"
)

type AuthMiddlewareTestSuite struct {
	suite.Suite
	access     *MockAccessService
	templates  *MockTemplates
	perms      httpserver.PermissionsBuilder
	authMiddle *auth.AuthMiddleware
}

type MockTemplates struct {
	mockLogin func(w http.ResponseWriter, errMsg string)
	mockError func(w http.ResponseWriter, errMsg string)
}

func (m *MockTemplates) Login(w http.ResponseWriter, errMsg string) {
	m.mockLogin(w, errMsg)
}

func (m *MockTemplates) LoginError(w http.ResponseWriter, errMsg string) {
	m.mockError(w, errMsg)
}

type MockAccessService struct {
	mockGetAccessKeyPermissions func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error)
	mockGenerateAccessKey       func(ctx context.Context, perms dto.AccessKeyPermissionsDto) (dto.AccessKeyPermissionsDto, error)
	mockDeleteAccessKeys        func(ctx context.Context, keys []string) error
}

func (m *MockAccessService) GetAccessKeyPermissions(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
	return m.mockGetAccessKeyPermissions(ctx, accessKey)
}

func (m *MockAccessService) GenerateAccessKey(ctx context.Context, perms dto.AccessKeyPermissionsDto) (dto.AccessKeyPermissionsDto, error) {
	return m.mockGenerateAccessKey(ctx, perms)
}

func (m *MockAccessService) DeleteAccessKeys(ctx context.Context, keys []string) error {
	return m.mockDeleteAccessKeys(ctx, keys)
}

func TestAuthMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(AuthMiddlewareTestSuite))
}

func (suite *AuthMiddlewareTestSuite) SetupTest() {
	suite.templates = &MockTemplates{}
	suite.access = &MockAccessService{}
	suite.perms = permissions.New()
	suite.authMiddle = auth.NewAuthMiddleware(suite.access, suite.perms, suite.templates)
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
				suite.access.mockGetAccessKeyPermissions = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
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
				suite.access.mockGetAccessKeyPermissions = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
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
				suite.access.mockGetAccessKeyPermissions = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
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
				suite.access.mockGetAccessKeyPermissions = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
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
				suite.access.mockGetAccessKeyPermissions = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
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
				suite.access.mockGetAccessKeyPermissions = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
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
				suite.access.mockGetAccessKeyPermissions = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
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
				suite.access.mockGetAccessKeyPermissions = func(ctx context.Context, accessKey string) (*permissions.AccessKeyPermissions, bool, error) {
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
