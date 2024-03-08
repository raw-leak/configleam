package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/raw-leak/configleam/internal/pkg/permissions"
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

func (r *RedisRepository) StoreAccessKey(ctx context.Context, accessKey AccessKey) error {
	perms, err := json.Marshal(accessKey.Perms)
	if err != nil {
		return err
	}

	bytePerms, err := r.encryptor.Encrypt(ctx, perms)
	if err != nil {
		return err
	}

	byteKey, err := r.encryptor.EncryptDet(ctx, []byte(accessKey.Key))
	if err != nil {
		return err
	}

	meta, err := json.Marshal(accessKey.Metadata)
	if err != nil {
		return err
	}
	currentTime := time.Now()
	var keyExpTime time.Duration

	if !accessKey.Metadata.ExpirationDate.IsZero() {
		keyExpTime = accessKey.Metadata.ExpirationDate.Sub(currentTime)
	} else {
		keyExpTime = 0
	}

	encryptedKey := base64.StdEncoding.EncodeToString(byteKey)
	pipeline := r.Client.TxPipeline()

	pipeline.Set(ctx, r.GetAccessKeyKey(encryptedKey), bytePerms, keyExpTime)
	pipeline.Set(ctx, r.GetAccessMetaKey(encryptedKey), meta, 0)
	pipeline.ZAdd(ctx, r.GetAccessSetKey(), redis.Z{Score: float64(accessKey.Metadata.CreationDate.Unix()), Member: encryptedKey})

	_, err = pipeline.Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisRepository) GetAccessKeyPermissions(ctx context.Context, key string) (*permissions.AccessKeyPermissions, bool, error) {
	encryptedKeyBytes, err := r.encryptor.EncryptDet(ctx, []byte(key))
	if err != nil {
		return nil, false, fmt.Errorf("error encrypting access-key '%s': %v", key, err)
	}

	key = base64.StdEncoding.EncodeToString(encryptedKeyBytes)

	raw, err := r.Client.Get(ctx, r.GetAccessKeyKey(key)).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("error getting key from redis '%s': %v", key, err)
	}

	raw, err = r.encryptor.Decrypt(ctx, raw)
	if err != nil {
		return nil, false, fmt.Errorf("error decrypting access-key permissions '%s': %v", key, err)
	}

	var perms permissions.AccessKeyPermissions
	err = json.Unmarshal(raw, &perms)
	if err != nil {
		return nil, false, fmt.Errorf("error unmarshalling access-key '%s': %v", key, err)
	}

	return &perms, true, nil
}

func (r *RedisRepository) PaginateAccessKeys(ctx context.Context, page int, size int) (*PaginatedAccessKeys, error) {
	accessKeysMetadata := []AccessKeyMetadata{}

	if size == 0 {
		size = 10
	}

	if page == 0 {
		size = 1
	}

	total, err := r.Client.ZCount(ctx, r.GetAccessSetKey(), "-inf", "+inf").Result()
	if err != nil {
		return nil, err
	}

	if total < 1 {
		return &PaginatedAccessKeys{Page: page, Size: size, Items: accessKeysMetadata, Total: 0, Pages: 0}, nil
	}

	keys, err := r.Client.ZRevRangeByScore(ctx, r.GetAccessSetKey(), &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: int64((page - 1) * size),
		Count:  int64(size),
	}).Result()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		metadataBytes, err := r.Client.Get(ctx, r.GetAccessMetaKey(key)).Bytes()
		if err != nil {
			return nil, err
		}

		var metadata AccessKeyMetadata
		err = json.Unmarshal(metadataBytes, &metadata)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling access key metadata '%s': %v", key, err)
		}

		metadata.Key = key
		accessKeysMetadata = append(accessKeysMetadata, metadata)
	}

	return &PaginatedAccessKeys{
			Page:  page,
			Size:  size,
			Items: accessKeysMetadata,
			Total: int(total),
			Pages: (int(total) + size - 1) / size,
		},
		nil
}

func (r *RedisRepository) RemoveKeys(ctx context.Context, keys []string) error {
	pipeline := r.Client.TxPipeline()

	for _, key := range keys {
		pipeline.Del(ctx, r.GetAccessKeyKey(key))
		pipeline.Del(ctx, r.GetAccessMetaKey(key))
		pipeline.ZRem(ctx, r.GetAccessSetKey(), key)
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

func (r *RedisRepository) GetAccessSetKey() string {
	return fmt.Sprintf("%s:%s", AccessPrefix, SetPrefix)
}
