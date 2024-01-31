package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/raw-leak/configleam/internal/app/configleam/types"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
	"github.com/redis/go-redis/v9"
)

type RedisConfigRepository struct {
	client *rds.Redis
}

func NewRedisConfigRepository(redis *rds.Redis) *RedisConfigRepository {
	return &RedisConfigRepository{redis}
}

func (r *RedisConfigRepository) StoreConfig(ctx context.Context, config *types.ParsedRepoConfig) error {
	pipeline := r.client.Client.TxPipeline()

	// Base key prefix based on environment and repository
	// Key: env:repo:configType:configName
	baseKeyPrefix := fmt.Sprintf("%s:%s:", config.EnvName, config.GitRepoName)

	// Store global configurations
	// Key: env:repo:global:configName
	for key, value := range config.Globals {
		globalKey := baseKeyPrefix + "global:" + key
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("error marshaling global config '%s': %v", key, err)
		}
		pipeline.Set(ctx, globalKey, jsonData, 0)
	}

	// Store group configurations
	// Key: env:repo:group:configName
	for groupName, groupConfig := range config.Groups {
		groupKey := baseKeyPrefix + "group:" + groupName
		jsonData, err := json.Marshal(groupConfig)
		if err != nil {
			return fmt.Errorf("error marshaling group config '%s': %v", groupName, err)
		}
		pipeline.Set(ctx, groupKey, jsonData, 0)
	}

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return fmt.Errorf("error executing Redis transaction: %v", err)
	}

	return nil
}

func (r *RedisConfigRepository) DeleteEnvConfigs(ctx context.Context, envName, gitRepoName string) error {
	luaScript := `
        local keys = redis.call('keys', ARGV[1])
        for i=1,#keys do
            redis.call('del', keys[i])
        end
        return #keys
    `

	keyPattern := fmt.Sprintf("%s:%s:*", envName, gitRepoName)
	result, err := r.client.Client.Eval(ctx, luaScript, []string{}, keyPattern).Result()
	if err != nil {
		return fmt.Errorf("error executing Lua script for deletion: %v", err)
	}

	log.Printf("Deleted %d keys matching the pattern '%s'", result, keyPattern)
	return nil
}

func (r *RedisConfigRepository) ReadConfig(ctx context.Context, groups []string, globalKeys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, groupName := range groups {
		groupKey := fmt.Sprintf("group:%s", groupName)
		val, err := r.client.Client.Get(ctx, groupKey).Result()

		if err == redis.Nil {
			// group key does not exist, skip
			continue
		} else if err != nil {
			return nil, fmt.Errorf("error fetching group config '%s': %v", groupName, err)
		}

		var groupConfig types.GroupConfig
		err = json.Unmarshal([]byte(val), &groupConfig)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling group config '%s': %v", groupName, err)
		}

		// combine local and referenced global configurations for the group
		combinedGroupConfig := make(map[string]interface{})
		for _, globalKey := range groupConfig.Global {
			globalVal, ok := result[globalKey]
			if !ok {
				// fetch global value if not already fetched
				fetchedVal, fetchErr := r.client.Client.Get(ctx, fmt.Sprintf("global:%s", globalKey)).Result()
				if fetchErr != nil && fetchErr != redis.Nil {
					return nil, fmt.Errorf("error fetching global config '%s': %v", globalKey, fetchErr)
				}

				json.Unmarshal([]byte(fetchedVal), &globalVal)
				result[globalKey] = globalVal
			}
			combinedGroupConfig[globalKey] = globalVal
		}

		for localKey, localVal := range groupConfig.Local {
			combinedGroupConfig[localKey] = localVal
		}

		result[groupName] = combinedGroupConfig
	}

	// Read additional global keys
	for _, key := range globalKeys {
		if _, ok := result[key]; !ok {
			val, err := r.client.Client.Get(ctx, fmt.Sprintf("global:%s", key)).Result()
			if err == redis.Nil {
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
