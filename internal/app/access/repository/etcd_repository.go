package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/raw-leak/configleam/internal/pkg/etcd"
	"github.com/raw-leak/configleam/internal/pkg/permissions"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdRepository struct {
	*etcd.Etcd
	encryptor Encryptor
}

func NewEtcdRepository(redis *etcd.Etcd, encryptor Encryptor) *EtcdRepository {
	return &EtcdRepository{redis, encryptor}
}

func (r *EtcdRepository) StoreAccessKey(ctx context.Context, accessKey AccessKey) error {
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
	encryptedKey := base64.StdEncoding.EncodeToString(byteKey)

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

	ops := make([]clientv3.Op, 0, 2)

	// access-key
	if keyExpTime > 0 {
		lease, err := r.Client.Grant(ctx, int64(keyExpTime.Seconds()))
		if err != nil {
			return err
		}

		ops = append(ops, clientv3.OpPut(r.GetAccessKeyKey(encryptedKey), string(bytePerms), clientv3.WithLease(lease.ID)))
	} else {
		ops = append(ops, clientv3.OpPut(r.GetAccessKeyKey(encryptedKey), string(bytePerms)))
	}

	// meta
	ops = append(ops, clientv3.OpPut(r.GetAccessMetaKey(encryptedKey), string(meta)))

	txnResp, err := r.Client.Txn(ctx).Then(ops...).Commit()
	if err != nil {
		return fmt.Errorf("error executing etcd transaction on storing config: %v", err)
	}
	if !txnResp.Succeeded {
		return fmt.Errorf("etcd transaction failed")
	}

	return nil
}

func (r *EtcdRepository) GetAccessKeyPermissions(ctx context.Context, key string) (*permissions.AccessKeyPermissions, bool, error) {
	encryptedKeyBytes, err := r.encryptor.EncryptDet(ctx, []byte(key))
	if err != nil {
		return nil, false, fmt.Errorf("error encrypting access-key '%s': %v", key, err)
	}

	key = base64.StdEncoding.EncodeToString(encryptedKeyBytes)

	res, err := r.Client.Get(ctx, r.GetAccessKeyKey(key))
	if len(res.Kvs) < 1 {
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("error getting key from redis '%s': %v", key, err)
	}

	dec, err := r.encryptor.Decrypt(ctx, res.Kvs[0].Value)
	if err != nil {
		return nil, false, fmt.Errorf("error decrypting access-key permissions '%s': %v", key, err)
	}

	var perms permissions.AccessKeyPermissions
	err = json.Unmarshal(dec, &perms)
	if err != nil {
		return nil, false, fmt.Errorf("error unmarshalling access-key '%s': %v", key, err)
	}

	return &perms, true, nil
}

func (r *EtcdRepository) PaginateAccessKeys(ctx context.Context, page int, size int) (*PaginatedAccessKeys, error) {
	accessKeysMetadata := []AccessKeyMetadata{}

	if size < 1 {
		size = 10
	}

	if page < 1 {
		size = 1
	}

	res, err := r.Client.Get(ctx, r.GetAccessMetaKey(""), clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortAscend))
	if err != nil {
		return nil, err
	}
	allKeys := res.Kvs

	if len(allKeys) < 1 {
		return &PaginatedAccessKeys{Page: page, Size: size, Items: accessKeysMetadata, Total: 0, Pages: 0}, nil
	}

	from := (page - 1) * size
	to := page * size

	if to > len(allKeys) {
		to = len(allKeys)
	}

	if from > len(allKeys) {
		from = len(allKeys)
	}

	pageKeys := allKeys[from:to]

	ops := make([]clientv3.Op, 0, len(pageKeys))

	for _, kv := range pageKeys {
		ops = append(ops, clientv3.OpGet(string(kv.Key)))
	}

	txnResp, err := r.Client.Txn(ctx).Then(ops...).Commit()
	if err != nil {
		return nil, fmt.Errorf("error executing etcd transaction on reading access keys by page: %v", err)
	}
	if !txnResp.Succeeded {
		return nil, fmt.Errorf("etcd transaction failed")
	}

	for _, opResp := range txnResp.Responses {
		res := opResp.GetResponseRange()
		key := string(res.Kvs[0].Key)

		var metadata AccessKeyMetadata
		err = json.Unmarshal(res.Kvs[0].Value, &metadata)
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
			Total: len(allKeys),
			Pages: (len(allKeys) + size - 1) / size,
		},
		nil
}

func (r *EtcdRepository) RemoveKeys(ctx context.Context, keys []string) error {
	ops := make([]clientv3.Op, 0, len(keys)*2)

	for _, key := range keys {
		// access-key
		ops = append(ops, clientv3.OpDelete(r.GetAccessKeyKey(key)))
		// meta
		ops = append(ops, clientv3.OpDelete(r.GetAccessMetaKey(key)))
	}

	txnResp, err := r.Client.Txn(ctx).Then(ops...).Commit()
	if err != nil {
		return fmt.Errorf("error executing etcd transaction on deleting access keys: %v", err)
	}
	if !txnResp.Succeeded {
		return fmt.Errorf("etcd transaction failed")
	}

	return nil
}

func (r *EtcdRepository) GetAccessKeyKey(key string) string {
	return fmt.Sprintf("%s:%s:%s", AccessPrefix, KeyPrefix, key)
}

func (r *EtcdRepository) GetAccessMetaKey(key string) string {
	return fmt.Sprintf("%s:%s:%s", AccessPrefix, MetaPrefix, key)
}

func (r *EtcdRepository) GetAccessSetKey() string {
	return fmt.Sprintf("%s:%s", AccessPrefix, SetPrefix)
}
