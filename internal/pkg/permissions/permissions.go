package permissions

type Operation uint

const (
	Admin            Operation = 1 << iota // 1
	EnvAdminAccess                         // 2
	ReadConfig                             // 4
	RevealSecrets                          // 8
	CloneEnvironment                       // 16
	CreateSecrets                          // 32
	AccessDashboard                        // 64
)

// AccessKeyPermissionsBuilder allows building new Access Key Permissions
type AccessKeyPermissionsBuilder struct{}

type AvailableAccessKeyPermissions struct {
	EnvAdminAccess   bool `json:"envAdminAccess"`
	ReadConfig       bool `json:"readConfig"`
	RevealSecrets    bool `json:"revealSecrets"`
	CloneEnvironment bool `json:"cloneEnvironment"`
	CreateSecrets    bool `json:"createSecrets"`
	AccessDashboard  bool `json:"accessDashboard"`
}

func New() *AccessKeyPermissionsBuilder {
	return &AccessKeyPermissionsBuilder{}
}

// NewAccessKeyPermissions initializes a AccessKeyPermissions struct with default values.
func (b AccessKeyPermissionsBuilder) NewAccessKeyPermissions() *AccessKeyPermissions {
	return &AccessKeyPermissions{
		Permissions: make(Permissions),
	}
}

type SinglePermission struct {
	Label   string
	Tooltip string
	Value   string
}

func (b AccessKeyPermissionsBuilder) GetAvailableAccessKeyPermissions() []SinglePermission {
	return []SinglePermission{
		{Label: "Environment Administrator", Tooltip: "Grants full administrative access to the specific environment.", Value: "envAdminAccess"},
		{Label: "Read Configuration", Tooltip: "Grants permission to view the configuration settings of the specific environment.", Value: "readConfig"},
		{Label: "Reveal Configuration Secrets", Tooltip: "Grants permission to reveal secrets within the configuration settings of the specific environment.", Value: "revealSecrets"},
		{Label: "Clone Environment Configuration", Tooltip: "Grants permission to create a new configuration by duplicating an existing one and adjusting certain global parameters.", Value: "cloneEnvironment"},
		{Label: "Create Environment Secrets", Tooltip: "Grants permission to create secrets for the specific environment.", Value: "createSecrets"},
		{Label: "Access Dashboard", Tooltip: "Grants permission to access the administrative dashboard for the specific environment.", Value: "accessDashboard"},
	}
}

// Permissions maps environments to their respective operation permissions.
type Permissions map[string]Operation

// AccessKeyPermissions holds the overall permissions structure for a user,
// including a special admin flag for overarching permissions.
type AccessKeyPermissions struct {
	Admin       bool
	Permissions Permissions
}

// Grant grants specified operations to a user for a given environment.
func (up *AccessKeyPermissions) Grant(env string, ops Operation) {
	if up.Permissions == nil {
		up.Permissions = make(Permissions)
	}
	up.Permissions[env] |= ops
}

// Can checks if a user has permissions to perform specified operations in a given environment.
func (up *AccessKeyPermissions) Can(env string, ops Operation) bool {
	if up.Admin {
		return true
	}

	currentOps, ok := up.Permissions[env]
	if !ok {
		return false
	}

	if currentOps&EnvAdminAccess == EnvAdminAccess {
		return true
	}

	return currentOps&ops == ops
}

// IsGlobalAdmin checks if a user has permission of global admin.
func (up *AccessKeyPermissions) IsGlobalAdmin() bool {
	return up.Admin
}

// SetAdmin grants admin permissions.
func (up *AccessKeyPermissions) SetAdmin() {
	up.Admin = true
}

func (up *AccessKeyPermissions) CanRevealSecrets(env string) bool {
	if up.Admin {
		return true
	}

	currentOps, ok := up.Permissions[env]
	if !ok {
		return false
	}

	if currentOps&EnvAdminAccess == EnvAdminAccess {
		return true
	}

	return currentOps&RevealSecrets == RevealSecrets
}
