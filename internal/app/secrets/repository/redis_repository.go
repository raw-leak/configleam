package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	rds "github.com/raw-leak/configleam/internal/pkg/redis"
	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	*rds.Redis
	encryptor Encryptor
}

func NewRedisRepository(redis *rds.Redis, encryptor Encryptor) *RedisRepository {
	return &RedisRepository{redis, encryptor}
}

func (r *RedisRepository) GetSecret(ctx context.Context, env, fullKey string) (interface{}, error) {
	keyPath := strings.Split(fullKey, ".")
	if len(keyPath) < 1 {
		return nil, fmt.Errorf("the key '%s' is malformed", fullKey)
	}

	key := keyPath[0]
	encryptedBytes, err := r.Client.Get(ctx, r.GetSecretKey(env, key)).Bytes()
	if err == redis.Nil {
		log.Printf("key '%s' was not found", fullKey)
		return nil, SecretNotFoundError{Key: fullKey}

	}
	if err != nil {
		return nil, err
	}

	decryptedBytes, err := r.encryptor.Decrypt(ctx, encryptedBytes)
	if err != nil {
		return nil, err
	}

	var value interface{}
	err = json.Unmarshal(decryptedBytes, &value)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling secret '%s': %v", key, err)
	}

	if len(keyPath) == 1 {
		return value, nil
	}

	if nested, ok := value.(map[string]interface{}); ok {
		value, ok := r.getValueByNestedKeys(nested, keyPath[1:])
		if ok {
			return value, nil
		} else {
			return nil, SecretNotFoundError{Key: fullKey}
		}
	}

	return nil, nil
}

func (r *RedisRepository) UpsertSecrets(ctx context.Context, env string, secrets map[string]interface{}) error {
	if len(secrets) < 1 {
		return errors.New("provided configuration is empty")
	}

	for key, value := range secrets {
		if strings.Contains(key, ".") {
			return fmt.Errorf("the secret configuration key '%s' is malformed", key)
		}

		if value == nil {
			return fmt.Errorf("'nil' can not be used as value for the key '%s'", key)
		}
	}

	pipeline := r.Client.TxPipeline()

	for key, value := range secrets {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		encrypted, err := r.encryptor.Encrypt(ctx, data)
		if err != nil {
			return err
		}

		err = pipeline.Set(ctx, r.GetSecretKey(env, key), encrypted, 0).Err()
		if err != nil {
			return err
		}
	}

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return fmt.Errorf("error executing Redis transaction on storing secrets config: %v", err)
	}

	return nil
}

func (r *RedisRepository) GetSecretKey(env, key string) string {
	return fmt.Sprintf("%s:%s:%s", SecretPrefix, env, key)
}

func (r *RedisRepository) getValueByNestedKeys(m map[string]interface{}, keys []string) (interface{}, bool) {
	var val interface{} = m

	for _, key := range keys {
		v := reflect.ValueOf(val)
		if v.Kind() != reflect.Map {
			return nil, false
		}

		val = v.MapIndex(reflect.ValueOf(key)).Interface()
		if val == nil {
			return nil, false
		}
	}

	return val, true
}

func (r *RedisRepository) CloneSecrets(ctx context.Context, cloneEnv, newEnv string) error {
	matchPattern := r.GetCloneSecretsPatternKey(cloneEnv)
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
		err = r.DeleteSecrets(ctx, newEnv)
		log.Fatalf("Error executing Lua script: %v", err)
		return err
	} else {
		log.Println("Keys duplicated successfully.")
	}

	return nil
}

func (r *RedisRepository) DeleteSecrets(ctx context.Context, env string) error {
	luaScript := `
        local keys = redis.call('keys', ARGV[1])
        for i=1,#keys do
            redis.call('del', keys[i])
        end
        return #keys
    `

	keyPattern := r.GetCloneSecretsDeletePatternKey(env)
	result, err := r.Client.Eval(ctx, luaScript, []string{}, keyPattern).Result()
	if err != nil {
		return fmt.Errorf("error executing Lua script for secret deletion: %v", err)
	}

	log.Printf("Deleted %d secret keys matching the pattern '%s'", result, keyPattern)
	return nil
}

func (r *RedisRepository) GetCloneSecretsPatternKey(cloneEnv string) string {
	return fmt.Sprintf("%s:%s:*", SecretPrefix, cloneEnv)
}

func (r *RedisRepository) GetCloneSecretsDeletePatternKey(clonedEnv string) string {
	return fmt.Sprintf("%s:*:%s:*", SecretPrefix, clonedEnv)
}
