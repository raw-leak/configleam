package types

type ExtractedConfigList []map[string]interface{}

type GroupConfig struct {
	// need to store all the local key-value of the group
	Local map[string]interface{}
	// need to store all the global keys of a group
	Global []string
}

type ParsedRepoConfig struct {
	// environment name
	EnvName string
	// gir repository name
	GitRepoName string
	// need to store all the groups
	Groups map[string]GroupConfig
	// need to store all the global variables
	Globals map[string]interface{}
	// need to store all the keys used in this file
	AllKeys []string
}
