package repository

import "fmt"


type RedisKeys struct{}

func (k RedisKeys) GetBaseKey(gitRepoName string, envName string) string {
	return fmt.Sprintf("%s:%s:%s", ConfigurationPrefix, gitRepoName, envName)
}

func (k RedisKeys) GetGlobalKeyKey(baseKeyPrefix string, GlobalPrefix string, configKey string) string {
	return fmt.Sprintf("%s:%s:%s", baseKeyPrefix, GlobalPrefix, configKey)
}

func (k RedisKeys) GetGroupKey(baseKeyPrefix string, groupName string) string {
	return fmt.Sprintf("%s:%s", baseKeyPrefix, groupName)
}

func (k RedisKeys) GetGroupPatternKey(env string, groupName string) string {
	return fmt.Sprintf("%s:*:%s:%s:%s", ConfigurationPrefix, env, GroupPrefix, groupName)
}

func (k RedisKeys) GetGlobalPatternKey(env string, key string) string {
	return fmt.Sprintf("%s:*:%s:%s:%s", ConfigurationPrefix, env, GlobalPrefix, key)
}

func (k RedisKeys) GetCloneEnvPatternKey(cloneEnv string) string {
	return fmt.Sprintf("%s:*:%s:*", ConfigurationPrefix, cloneEnv)
}

func (k RedisKeys) GetCloneEnvDeletePatternKey(gitRepoName, clonedEnv string) string {
	return fmt.Sprintf("%s:*:%s:%s:*", ConfigurationPrefix, gitRepoName, clonedEnv)
}

func (k RedisKeys) GetEnvKey(envName string) string {
	return fmt.Sprintf("%s:%s", ConfigurationEnvPrefix, envName)
}

func (k RedisKeys) GetAllEnvsPatternKey() string {
	return fmt.Sprintf("%s:*", ConfigurationEnvPrefix)
}

func (k RedisKeys) GetEnvLockKey(envName string) string {
	return fmt.Sprintf("%s:%s", ConfigurationLockPrefix, envName)
}
