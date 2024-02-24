package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/raw-leak/configleam/internal/app/configuration/types"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
	"github.com/redis/go-redis/v9"
)

const (
	GlobalPrefix          = "global"
	ReadLockMaxRetries    = 3
	ReadLockRetryInterval = 500 * time.Millisecond
)

type RedisConfigRepository struct {
	*rds.Redis
}

func NewRedisRepository(redis *rds.Redis) *RedisConfigRepository {
	return &RedisConfigRepository{redis}
}

func (r *RedisConfigRepository) storeConfig(ctx context.Context, envName string, gitRepoName string, config *types.ParsedRepoConfig) error {
	pipeline := r.Client.TxPipeline()

	// Base key prefix based on environment and repository
	// Key: <repo>:<env>:global|group:<configKey>
	baseKeyPrefix := fmt.Sprintf("%s:%s:", gitRepoName, envName)

	// Store global configurations
	// Key: <repo>:<env>:global:<key>
	for configKey, value := range config.Globals {
		globalKey := fmt.Sprintf("%s%s:%s", baseKeyPrefix, GlobalPrefix, configKey)
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("error marshaling global config '%s': %v", configKey, err)
		}
		pipeline.Set(ctx, globalKey, jsonData, 0)
	}

	// Store group configurations
	// Key: <repo>:<env>:group:<groupName>
	for groupName, groupConfig := range config.Groups {
		groupKey := baseKeyPrefix + groupName

		jsonData, err := json.Marshal(groupConfig)
		if err != nil {
			return fmt.Errorf("error marshaling group config '%s': %v", groupName, err)
		}

		pipeline.Set(ctx, groupKey, jsonData, 0)
	}

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return fmt.Errorf("error executing Redis transaction on storing config: %v", err)
	}

	return nil
}

// TODO: test
func (r *RedisConfigRepository) DeleteConfig(ctx context.Context, env, gitRepoName string) error {
	luaScript := `
        local keys = redis.call('keys', ARGV[1])
        for i=1,#keys do
            redis.call('del', keys[i])
        end
        return #keys
    `

	keyPattern := fmt.Sprintf("%s:%s:*", gitRepoName, env)
	result, err := r.Client.Eval(ctx, luaScript, []string{}, keyPattern).Result()
	if err != nil {
		return fmt.Errorf("error executing Lua script for deletion: %v", err)
	}

	log.Printf("Deleted %d keys matching the pattern '%s'", result, keyPattern)
	return nil
}

func (r *RedisConfigRepository) UpsertConfig(ctx context.Context, env string, gitRepoName string, config *types.ParsedRepoConfig) error {
	lockKey := fmt.Sprintf("lock:%s", env)

	_, err := r.Client.Set(ctx, lockKey, "lock", time.Second).Result()
	if err != nil {
		return fmt.Errorf("error acquiring lock: %v", err)
	}
	defer r.Client.Del(ctx, lockKey)

	err = r.DeleteConfig(ctx, env, gitRepoName)
	if err != nil {
		return fmt.Errorf("error acquiring lock: %v", err)
	}

	return r.storeConfig(ctx, env, gitRepoName, config)
}

func (r *RedisConfigRepository) ReadConfig(ctx context.Context, env string, groups, globalKeys []string) (map[string]interface{}, error) {
	err := r.checkLockAndRetry(ctx, env)
	if err != nil {
		return nil, fmt.Errorf("error verifying the lock while reading config for environment '%s': %v", env, err)
	}

	result := map[string]interface{}{}
	for _, groupName := range groups {
		// look for: *:<env>:group:<groupName>
		// returns provided group collection from any repository
		groupKeyPattern := fmt.Sprintf("*:%s:group:%s", env, groupName)

		// keys len could be equal to the amount of repositories connected to the configleam
		// IMP: there will be a small number of group keys
		groupKeys, err := r.Client.Keys(ctx, groupKeyPattern).Result()
		if err != nil {
			return nil, fmt.Errorf("error fetching keys for groups config '%s': %v", groupName, err)
		}

		groupsConfig := map[string]types.GroupConfig{}
		for _, groupKey := range groupKeys {
			var groupConfig types.GroupConfig

			val, err := r.Client.Get(ctx, groupKey).Result()
			if err == redis.Nil {
				// group's key does not exist, skip
				log.Printf("key '%s' was not found while reading config\n", groupKey)
				continue
			} else if err != nil {
				return nil, fmt.Errorf("error fetching group config '%s': %v", groupName, err)
			}

			err = json.Unmarshal([]byte(val), &groupConfig)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling group config '%s': %v", groupName, err)
			}

			groupsConfig[groupName] = groupConfig
		}

		// combine local and referenced global configurations for the group (goroutine?)
		combinedGroupConfig := map[string]interface{}{}
		for _, groupConfig := range groupsConfig {
			// IMP: there could be many local keys
			for localKey, localVal := range groupConfig.Local {
				combinedGroupConfig[localKey] = localVal
			}

			// IMP: there could be many global keys (goroutine?)
			for _, key := range groupConfig.Global {
				if _, ok := combinedGroupConfig[key]; !ok {
					// fetch global value if not already fetched

					globalKeyPattern := fmt.Sprintf("*:%s:global:%s", env, key)
					keys, err := r.Client.Keys(ctx, globalKeyPattern).Result()
					if err != nil {
						return nil, fmt.Errorf("error reading keys by pattern '%s': %v", globalKeyPattern, err)
					}
					if len(keys) < 1 {
						log.Printf("key '%s' was not found in '%s' environment while getting keys", key, env)
						continue
					}

					globalKey := keys[0]
					val, err := r.Client.Get(ctx, globalKey).Result()
					if err == redis.Nil {
						log.Printf("key '%s' was not found in '%s' environment while getting a single key for group '%s'", key, env, groupName)
						// global key does not exist, skip
						continue
					} else if err != nil {
						return nil, fmt.Errorf("error fetching global config '%s' for group '%s': %v", key, groupName, err)
					}

					var globalVal interface{}
					err = json.Unmarshal([]byte(val), &globalVal)
					if err != nil {
						return nil, fmt.Errorf("error unmarshalling global config '%s' for groups '%s': %v", key, groupName, err)
					}

					combinedGroupConfig[key] = globalVal
				}
			}
		}

		result[groupName] = combinedGroupConfig
	}

	// read additional global keys
	for _, key := range globalKeys {
		if _, ok := result[key]; !ok {
			// look for a global key with next pattern: *:env:global:key
			globalKeyPattern := fmt.Sprintf("*:%s:global:%s", env, key)
			keys, err := r.Client.Keys(ctx, globalKeyPattern).Result()
			if err != nil {
				return nil, fmt.Errorf("error reading keys by pattern '%s': %v", globalKeyPattern, err)
			}
			if len(keys) < 1 {
				log.Printf("key '%s' was not found in '%s' environment while getting keys", key, env)
				continue
			}

			globalKey := keys[0]
			val, err := r.Client.Get(ctx, globalKey).Result()
			if err == redis.Nil {
				log.Printf("key '%s' was not found in '%s' environment while getting a single key", key, env)
				// global key does not exist, skip
				continue
			} else if err != nil {
				return nil, fmt.Errorf("error fetching global config '%s': %v", key, err)
			}

			var globalVal interface{}
			err = json.Unmarshal([]byte(val), &globalVal)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling global config '%s': %v", key, err)
			}

			result[key] = globalVal
		}
	}

	return result, nil
}

func (r *RedisConfigRepository) checkLockAndRetry(ctx context.Context, env string) error {
	lockKey := fmt.Sprintf("lock:%s", env)

	for retry := 1; retry <= ReadLockMaxRetries; retry++ {
		// Check if the lock key exists
		exists, err := r.Client.Exists(ctx, lockKey).Result()
		if err != nil {
			return fmt.Errorf("error checking lock: %v", err)
		}

		if exists == 0 {
			return nil
		}

		if retry < ReadLockMaxRetries {
			time.Sleep(ReadLockRetryInterval)
		}
	}

	return errors.New("timeout waiting for the lock")
}

func (r *RedisConfigRepository) CloneConfig(ctx context.Context, cloneEnv, newEnv string, updateGlobal map[string]interface{}) error {
	matchPattern := fmt.Sprintf("*:%s:*", cloneEnv)
	oldSegment := fmt.Sprintf(":%s:", cloneEnv)
	newSegment := fmt.Sprintf(":%s:", newEnv)

	script := `local matchPattern = KEYS[1]
               local oldSegment = ARGV[1]
               local newSegment = ARGV[2]
               local cursor = "0"
               local done = false

               repeat
                   local result = redis.call("SCAN", cursor, "MATCH", matchPattern)
                   cursor = result[1]
                   local keys = result[2]

                   for i, key in ipairs(keys) do
                       local value = redis.call("GET", key)
                       local newKey = string.gsub(key, oldSegment, newSegment)
                       redis.call("SET", newKey, value)
                   end

                   if cursor == "0" then
                       done = true
                   end
               until done

               return "Keys duplicated successfully"`

	_, err := r.Client.Eval(ctx, script, []string{matchPattern}, oldSegment, newSegment).Result()
	if err != nil {
		err = r.DeleteConfig(ctx, newEnv, "*")
		log.Fatalf("Error executing Lua script: %v", err)
		return err
	} else {
		log.Println("Keys duplicated successfully.")
	}

	if len(updateGlobal) > 0 {
		pipeline := r.Client.Pipeline()

		for k, v := range updateGlobal {
			globalKeyMatchPattern := fmt.Sprintf("*:%s:%s:%s", cloneEnv, GlobalPrefix, k)

			globalKeys, err := r.Client.Keys(ctx, globalKeyMatchPattern).Result()
			if err != nil {
				delErr := r.DeleteConfig(ctx, newEnv, "*")
				if delErr != nil {
					// TODO LOG
				}
				return err
			}

			if len(globalKeys) > 0 {
				jsonData, err := json.Marshal(v)
				if err != nil {
					delErr := r.DeleteConfig(ctx, newEnv, "*")
					if delErr != nil {
						// TODO LOG
					}
					return err
				}

				for _, gk := range globalKeys {
					newEnvGk := strings.Replace(gk, oldSegment, newSegment, 1)
					pipeline.Set(ctx, newEnvGk, jsonData, 0)
				}
			}
		}

		_, err = pipeline.Exec(ctx)
		if err != nil {
			delErr := r.DeleteConfig(ctx, newEnv, "*")
			if delErr != nil {
				// TODO LOG
			}
			return err
		}
	}

	return nil
}

// TODO: test
func (r *RedisConfigRepository) HealthCheck(ctx context.Context) error {
	_, err := r.Client.Ping(ctx).Result()
	if err != nil {
		return err
	}

	return nil
}
