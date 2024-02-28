package dto_test

import (
	"testing"

	"github.com/raw-leak/configleam/internal/app/access/dto"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
	"github.com/stretchr/testify/suite"
)

type AccessKeyPermissionsDtoTestSuite struct {
	suite.Suite
}

func (suite *AccessKeyPermissionsDtoTestSuite) TestToAccessKeyPermissions() {
	tests := []struct {
		name     string
		dto      dto.AccessKeyPermissionsDto
		expected permissions.AccessKeyPermissions
	}{
		{
			name: "single environment with multiple permissions",
			dto: dto.AccessKeyPermissionsDto{
				Envs: map[string]dto.EnvironmentPermissions{
					"prod": {
						ReadConfig:       true,
						RevealSecrets:    true,
						CloneEnvironment: false,
						CreateSecrets:    true,
						AccessDashboard:  false,
					},
				},
			},
			expected: permissions.AccessKeyPermissions{
				Admin: false,
				Permissions: map[string]permissions.Operation{
					"prod": permissions.ReadConfig | permissions.RevealSecrets | permissions.CreateSecrets,
				},
			},
		},
		{
			name: "multiple environments with varied permissions",
			dto: dto.AccessKeyPermissionsDto{
				GlobalAdmin: true,
				Envs: map[string]dto.EnvironmentPermissions{
					"dev": {
						ReadConfig:       true,
						RevealSecrets:    false,
						CloneEnvironment: true,
						CreateSecrets:    false,
						AccessDashboard:  false,
					},
					"stage": {
						ReadConfig:       false,
						RevealSecrets:    true,
						CloneEnvironment: false,
						CreateSecrets:    true,
						AccessDashboard:  true,
					},
				},
			},
			expected: permissions.AccessKeyPermissions{
				Admin: true,
				Permissions: map[string]permissions.Operation{
					"dev":   permissions.ReadConfig | permissions.CloneEnvironment,
					"stage": permissions.RevealSecrets | permissions.CreateSecrets | permissions.AccessDashboard,
				},
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.Equal(tt.expected, tt.dto.ToAccessKeyPermissions())
		})
	}
}

func TestAccessKeyPermissionsDtoTestSuite(t *testing.T) {
	suite.Run(t, new(AccessKeyPermissionsDtoTestSuite))
}
