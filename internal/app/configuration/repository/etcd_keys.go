package repository

import "fmt"

type EtcdKeys struct{}

func (k EtcdKeys) GetReadGroupRepoEnvKey(repo, env, key string) string {
	return fmt.Sprintf("%s:%s:%s:%s:%s", ConfigurationPrefix, repo, env, GroupPrefix, key)
}

func (k EtcdKeys) GetReadGlobalRepoEnvKey(repo, env, key string) string {
	return fmt.Sprintf("%s:%s:%s:%s:%s", ConfigurationPrefix, repo, env, GlobalPrefix, key)
}

func (k EtcdKeys) GetBaseKey(repo string, env string) string {
	return fmt.Sprintf("%s:%s:%s", ConfigurationPrefix, repo, env)
}

func (k EtcdKeys) GetGlobalKey(prefix, key string) string {
	return fmt.Sprintf("%s:%s:%s", prefix, GlobalPrefix, key)
}

func (k EtcdKeys) GetCloneEnvPatternKey(cloneEnv string) string {
	return fmt.Sprintf("%s:*:%s:*", ConfigurationPrefix, cloneEnv)
}

func (k EtcdKeys) GetGroupKey(prefix, key string) string {
	return fmt.Sprintf("%s:%s:%s", prefix, GroupPrefix, key)
}

func (k EtcdKeys) GetEnvLockKey(env string) string {
	return fmt.Sprintf("%s:%s", ConfigurationLockPrefix, env)
}

func (k EtcdKeys) GetEnvKey(env string) string {
	return fmt.Sprintf("%s:%s", ConfigurationEnvPrefix, env)
}
