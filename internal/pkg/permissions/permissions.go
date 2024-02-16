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

func New() *AccessKeyPermissionsBuilder {
	return &AccessKeyPermissionsBuilder{}
}

// NewAccessKeyPermissions initializes a AccessKeyPermissions struct with default values.
func (b AccessKeyPermissionsBuilder) NewAccessKeyPermissions() *AccessKeyPermissions {
	return &AccessKeyPermissions{
		Permissions: make(Permissions),
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
		return true // Admins can do anything
	}
	currentOps, ok := up.Permissions[env]
	if !ok {
		return false // No permissions for this environment
	}
	return currentOps&ops == ops
}

// SetAdmin grants admin permissions.
func (up *AccessKeyPermissions) SetAdmin() {
	up.Admin = true
}
