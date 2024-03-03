package repository

import "fmt"

const (
	ConfigurationPrefix    = "configleam:config"
	ConfigurationEnvPrefix = "configleam:env"
)

func (r RedisRepository) GetBaseKey(gitRepoName string, envName string) string {
	return fmt.Sprintf("%s:%s:%s", ConfigurationPrefix, gitRepoName, envName)
}

func (r RedisRepository) GetGlobalKeyKey(baseKeyPrefix string, GlobalPrefix string, configKey string) string {
	return fmt.Sprintf("%s:%s:%s", baseKeyPrefix, GlobalPrefix, configKey)
}

func (r RedisRepository) GetGroupKey(baseKeyPrefix string, groupName string) string {
	return fmt.Sprintf("%s:%s", baseKeyPrefix, groupName)
}

func (r RedisRepository) GetGroupPatternKey(env string, groupName string) string {
	return fmt.Sprintf("%s:*:%s:%s:%s", ConfigurationPrefix, env, GroupPrefix, groupName)
}

func (r RedisRepository) GetGlobalPatternKey(env string, key string) string {
	return fmt.Sprintf("%s:*:%s:%s:%s", ConfigurationPrefix, env, GlobalPrefix, key)
}

func (r RedisRepository) GetCloneEnvPatternKey(cloneEnv string) string {
	return fmt.Sprintf("%s:*:%s:*", ConfigurationPrefix, cloneEnv)
}

func (r RedisRepository) GetCloneEnvDeletePatternKey(gitRepoName, clonedEnv string) string {
	return fmt.Sprintf("%s:*:%s:%s:*", ConfigurationPrefix, gitRepoName, clonedEnv)
}

func (r RedisRepository) GetEnvKey(envName string) string {
	return fmt.Sprintf("%s:%s", ConfigurationEnvPrefix, envName)
}

func (r RedisRepository) GetAllEnvsPatternKey() string {
	return fmt.Sprintf("%s:*", ConfigurationEnvPrefix)
}
