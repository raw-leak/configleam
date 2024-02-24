package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	rds "github.com/raw-leak/configleam/internal/pkg/redis"
	"github.com/redis/go-redis/v9"
)

const (
	SecretPrefix = "configleam:secret:"
)

type Encryptor interface {
	Encrypt(ctx context.Context, b []byte) ([]byte, error)
	Decrypt(ctx context.Context, b []byte) ([]byte, error)
}

type RedisRepository struct {
	*rds.Redis
	encryptor Encryptor
}

func NewRedisRepository(redis *rds.Redis, encryptor Encryptor) *RedisRepository {
	return &RedisRepository{redis, encryptor}
}

// GetSecret retrieves a secret from Redis.
func (r *RedisRepository) GetSecret(ctx context.Context, env, fullKey string) (interface{}, error) {
	keyPath := strings.Split(fullKey, ".")
	if len(keyPath) < 1 {
		return nil, fmt.Errorf("the key '%s' is malformed", fullKey)
	}

	key := keyPath[0]
	encryptedBytes, err := r.Client.Get(ctx, r.GetSecretKey(env, key)).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("key '%s' was not found", fullKey)

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
			return nil, fmt.Errorf("not found value for secret '%s'", fullKey)
		}
	}

	return nil, nil
}

// UpsertSecret stores a secret in Redis.
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

// UpsertSecret stores a secret in Redis.
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

func (r *RedisRepository) HealthCheck(ctx context.Context) error {
	_, err := r.Client.Ping(ctx).Result()
	if err != nil {
		return err
	}

	return nil
}
