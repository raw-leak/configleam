package dto

import "github.com/raw-leak/configleam/internal/pkg/permissions"

type AccessKeyPermissionsMeta struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// EnvironmentPermissions represents permissions for a single environment.
type EnvironmentPermissions struct {
	EnvAdminAccess   bool `json:"envAdminAccess"`
	ReadConfig       bool `json:"readConfig"`
	RevealSecrets    bool `json:"revealSecrets"`
	CloneEnvironment bool `json:"cloneEnvironment"`
	CreateSecrets    bool `json:"createSecrets"`
	AccessDashboard  bool `json:"accessDashboard"`
}

// AccessKeyPermissionsDto represents the permissions request for multiple environments.
type AccessKeyPermissionsDto struct {
	GlobalAdmin bool                              `json:"globalAdmin"`
	Envs        map[string]EnvironmentPermissions `json:"environments"`
	AccessKey   string                            `json:"accessKey,omitempty"`
	AccessKeyPermissionsMeta
}

func (req *AccessKeyPermissionsDto) ToAccessKeyPermissions() permissions.AccessKeyPermissions {
	perms := permissions.AccessKeyPermissions{}
	envOps := make(map[string]permissions.Operation)

	for env, perms := range req.Envs {
		var ops permissions.Operation
		if perms.ReadConfig {
			ops |= permissions.ReadConfig
		}
		if perms.RevealSecrets {
			ops |= permissions.RevealSecrets
		}
		if perms.CloneEnvironment {
			ops |= permissions.CloneEnvironment
		}
		if perms.CreateSecrets {
			ops |= permissions.CreateSecrets
		}
		if perms.AccessDashboard {
			ops |= permissions.AccessDashboard
		}
		if perms.EnvAdminAccess {
			ops |= permissions.EnvAdminAccess
		}
		envOps[env] = ops
	}

	perms.Permissions = envOps
	perms.Admin = req.GlobalAdmin

	return perms
}

func (req *AccessKeyPermissionsDto) ToMeta() map[string]string {
	return map[string]string{"name": req.Name, "description": req.Description}
}
