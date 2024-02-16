package permissions_test

import (
	"testing"

	"github.com/raw-leak/configleam/internal/pkg/permissions"
	"github.com/stretchr/testify/suite"
)

type AccessKeyPermissionsTestSuite struct {
	suite.Suite
}

func TestUserPermissionsTestSuite(t *testing.T) {
	suite.Run(t, new(AccessKeyPermissionsTestSuite))
}

func (suite *AccessKeyPermissionsTestSuite) TestPermissions() {
	tests := []struct {
		name           string
		setupFunc      func(up *permissions.AccessKeyPermissions)
		env            string
		operation      permissions.Operation
		expectedResult bool
	}{
		{
			name: "Grant ReadConfig permission and check",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.Grant("dev", permissions.ReadConfig)
			},
			env:            "dev",
			operation:      permissions.ReadConfig,
			expectedResult: true,
		},
		{
			name: "Check ReadConfig permission without granting",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				// No permissions granted
			},
			env:            "dev",
			operation:      permissions.ReadConfig,
			expectedResult: false,
		},
		{
			name: "Check ReadConfig permission granting CloneEnvironment",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.Grant("dev", permissions.CloneEnvironment)
			},
			env:            "dev",
			operation:      permissions.ReadConfig,
			expectedResult: false,
		},
		{
			name: "Admin permissions override",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.SetAdmin()
			},
			env:            "prod",
			operation:      permissions.CloneEnvironment,
			expectedResult: true,
		},
		{
			name: "Grant multiple permissions and check combined",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.Grant("prod", permissions.ReadConfig|permissions.CreateSecrets)
			},
			env:            "prod",
			operation:      permissions.ReadConfig | permissions.CreateSecrets,
			expectedResult: true,
		},
		{
			name: "Grant permissions in multiple environments",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.Grant("dev", permissions.ReadConfig)
				up.Grant("prod", permissions.CloneEnvironment)
			},
			env:            "dev",
			operation:      permissions.ReadConfig,
			expectedResult: true,
		},
		{
			name: "Permissions not granted in another environment",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.Grant("dev", permissions.ReadConfig)
			},
			env:            "prod",
			operation:      permissions.ReadConfig,
			expectedResult: false,
		},
		{
			name: "Combined permissions grant access",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.Grant("dev", permissions.ReadConfig|permissions.CloneEnvironment)
			},
			env:            "dev",
			operation:      permissions.ReadConfig | permissions.CloneEnvironment,
			expectedResult: true,
		},
		{
			name: "Partial permissions do not grant access",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.Grant("dev", permissions.ReadConfig)
			},
			env:            "dev",
			operation:      permissions.ReadConfig | permissions.CloneEnvironment,
			expectedResult: false,
		},
		{
			name: "Grant all operations except one",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.Grant("dev", permissions.ReadConfig|permissions.RevealSecrets|permissions.CreateSecrets) // Excluding permissions.CloneEnvironment
			},
			env:            "dev",
			operation:      permissions.CloneEnvironment,
			expectedResult: false,
		},
		{
			name: "No permissions in empty environment",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.Grant("dev", permissions.ReadConfig)
			},
			env:            "",
			operation:      permissions.ReadConfig,
			expectedResult: false,
		},
		{
			name: "Admin with no explicit permissions in environment",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.SetAdmin()
			},
			env:            "missing-env",
			operation:      permissions.AccessDashboard,
			expectedResult: true,
		},
		{
			name: "Grant and check permissions for non-existent operation",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				up.Grant("dev", permissions.Operation(128)) // Non-existent operation
			},
			env:            "dev",
			operation:      permissions.Operation(128),
			expectedResult: true,
		},
		{
			name: "Check without granting any permissions",
			setupFunc: func(up *permissions.AccessKeyPermissions) {
				// No permissions granted explicitly
			},
			env:            "any-env",
			operation:      permissions.ReadConfig,
			expectedResult: false,
		},
	}

	builder := permissions.New()

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			userPermissions := builder.NewAccessKeyPermissions()

			tc.setupFunc(userPermissions)

			suite.Equal(tc.expectedResult, userPermissions.Can(tc.env, tc.operation))
		})
	}
}
