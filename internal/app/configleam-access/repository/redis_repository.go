package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/raw-leak/configleam/internal/pkg/permissions"
	rds "github.com/raw-leak/configleam/internal/pkg/redis"
	"github.com/redis/go-redis/v9"
)

const (
	AccessPrefix = "configleam:access"
	KeyPrefix    = "key"
	MetaPrefix   = "meta"
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

func (r *RedisRepository) StoreKeyWithPermissions(ctx context.Context, key string, perms permissions.AccessKeyPermissions, meta map[string]string) error {
	data, err := json.Marshal(perms)
	if err != nil {
		return err
	}

	encrypted, err := r.encryptor.Encrypt(ctx, data)
	if err != nil {
		return err
	}

	pipeline := r.Client.TxPipeline()

	pipeline.Set(ctx, r.GetAccessKeyKey(key), encrypted, 0)

	if len(meta) > 0 {
		values := make([]interface{}, 0, len(meta)*2)
		for key, value := range meta {
			values = append(values, key, value)
		}

		pipeline.HSet(ctx, r.GetAccessMetaKey(key), values...)
	}

	_, err = pipeline.Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisRepository) GetKeyPermissions(ctx context.Context, key string) (*permissions.AccessKeyPermissions, bool, error) {
	encryptedBytes, err := r.Client.Get(ctx, r.GetAccessKeyKey(key)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("error getting key from redis '%s': %v", key, err)
	}

	decryptedBytes, err := r.encryptor.Decrypt(ctx, encryptedBytes)
	if err != nil {
		return nil, false, fmt.Errorf("error decrypting access-key permissions '%s': %v", key, err)
	}

	var perms permissions.AccessKeyPermissions
	err = json.Unmarshal(decryptedBytes, &perms)
	if err != nil {
		return nil, false, fmt.Errorf("error unmarshalling access-key '%s': %v", key, err)
	}

	return &perms, true, nil
}

func (r *RedisRepository) RemoveKeys(ctx context.Context, keys []string) error {
	pipeline := r.Client.TxPipeline()

	for _, key := range keys {
		pipeline.Del(ctx, r.GetAccessKeyKey(key))
	}

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisRepository) GetAccessKeyKey(key string) string {
	return fmt.Sprintf("%s:%s:%s", AccessPrefix, KeyPrefix, key)
}

func (r *RedisRepository) GetAccessMetaKey(key string) string {
	return fmt.Sprintf("%s:%s:%s", AccessPrefix, MetaPrefix, key)
}

func (r *RedisRepository) HealthCheck(ctx context.Context) error {
	_, err := r.Client.Ping(ctx).Result()
	if err != nil {
		return err
	}

	return nil
}
