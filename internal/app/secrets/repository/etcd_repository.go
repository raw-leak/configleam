package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/raw-leak/configleam/internal/pkg/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdRepository struct {
	*etcd.Etcd
	encryptor Encryptor
}

func NewEtcdRepository(etcd *etcd.Etcd, encryptor Encryptor) *EtcdRepository {
	return &EtcdRepository{etcd, encryptor}
}

func (r *EtcdRepository) GetSecret(ctx context.Context, env, fullKey string) (interface{}, error) {
	keyPath := strings.Split(fullKey, ".")
	if len(keyPath) < 1 {
		return nil, fmt.Errorf("the key '%s' is malformed", fullKey)
	}

	key := keyPath[0]
	res, err := r.Client.Get(ctx, r.GetSecretKey(env, key))
	if err != nil {
		return nil, err
	}
	if len(res.Kvs) < 1 {
		log.Printf("key '%s' was not found", fullKey)
		return nil, SecretNotFoundError{Key: fullKey}
	}

	decryptedBytes, err := r.encryptor.Decrypt(ctx, res.Kvs[0].Value)
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

func (r *EtcdRepository) UpsertSecrets(ctx context.Context, env string, secrets map[string]interface{}) error {
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

	ops := make([]clientv3.Op, 0, len(secrets))
	for key, value := range secrets {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		encrypted, err := r.encryptor.Encrypt(ctx, data)
		if err != nil {
			return err
		}

		ops = append(ops, clientv3.OpPut(r.GetSecretKey(env, key), string(encrypted)))
	}

	txnResp, err := r.Client.Txn(ctx).Then(ops...).Commit()
	if err != nil {
		return fmt.Errorf("error executing etcd transaction on storing secrets config: %v", err)
	}
	if !txnResp.Succeeded {
		return fmt.Errorf("etcd transaction failed")
	}

	return nil
}

func (r *EtcdRepository) CloneSecrets(ctx context.Context, env, newEnv string) error {
	prefix := r.GetBaseKey(env)

	oldSegment := fmt.Sprintf(":%s:", env)
	newSegment := fmt.Sprintf(":%s:", newEnv)

	key := prefix
	var lastErr error

MainLoop:
	for {
		res, err := r.Client.Get(ctx, key, clientv3.WithRange(clientv3.GetPrefixRangeEnd(prefix)), clientv3.WithLimit(25))
		if err != nil {
			return fmt.Errorf("failed to fetch keys: %v", err)
		}

		ops := make([]clientv3.Op, 0, len(res.Kvs))

		for _, kv := range res.Kvs {
			key := string(kv.Key)
			keyRes, err := r.Client.Get(ctx, key)
			if err != nil {
				lastErr = err
				break MainLoop
			}

			newKey := strings.Replace(key, oldSegment, newSegment, 1)
			ops = append(ops, clientv3.OpPut(newKey, string(keyRes.Kvs[0].Value)))
		}

		if len(ops) > 0 {
			_, txnErr := r.Client.Txn(ctx).Then(ops...).Commit()
			if txnErr != nil {
				log.Printf("failed to execute transaction for storing cloned secrets: %v", txnErr)
				lastErr = txnErr
			}
		}

		if !res.More {
			break
		}

		lastKey := res.Kvs[len(res.Kvs)-1].Key
		key = string(append(lastKey, 0))
	}

	if lastErr != nil {
		err := r.DeleteSecrets(ctx, newEnv)
		log.Printf("error cleaning the cloned '%s' environment secrets from '%s' environment secrets: %v", newEnv, env, err)
		return err
	}

	return nil
}

// DeleteSecrets deletes all secrets keys for a specific environment.
func (r *EtcdRepository) DeleteSecrets(ctx context.Context, env string) error {
	prefix := r.GetBaseKey(env)
	_, err := r.Client.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("error deleting secrets for '%s' environment: %v", env, err)
	}
	return nil
}

func (r *EtcdRepository) GetBaseKey(env string) string {
	return fmt.Sprintf("%s:%s", SecretPrefix, env)
}

func (r *EtcdRepository) GetSecretKey(env, key string) string {
	return fmt.Sprintf("%s:%s:%s", SecretPrefix, env, key)
}

func (r *EtcdRepository) getValueByNestedKeys(m map[string]interface{}, keys []string) (interface{}, bool) {
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
