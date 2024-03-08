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
	"github.com/raw-leak/configleam/internal/pkg/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdRepository struct {
	*etcd.Etcd
	keys EtcdKeys
}

func NewEtcdRepository(etcd *etcd.Etcd) *EtcdRepository {
	return &EtcdRepository{etcd, EtcdKeys{}}
}

func (r *EtcdRepository) UpsertConfig(ctx context.Context, repo, env string, config *types.ParsedRepoConfig) error {
	lockID, err := r.lockEnv(ctx, env)
	if err != nil {
		return fmt.Errorf("error locking '%s' environment for '%s' repository: %v", env, repo, err)
	}
	defer r.releaseEnvLock(ctx, lockID)

	err = r.DeleteConfig(ctx, repo, env)
	if err != nil {
		return fmt.Errorf("error deleting current '%s' environment configuration for '%s' repository: %v", env, repo, err)
	}

	return r.storeConfig(ctx, env, repo, config)
}

func (r *EtcdRepository) storeConfig(ctx context.Context, env, repo string, config *types.ParsedRepoConfig) error {
	basePrefix := r.keys.GetBaseKey(repo, env)

	ops := make([]clientv3.Op, 0, len(config.Globals)+len(config.Groups))

	for globalName, globalVal := range config.Globals {
		globalKey := r.keys.GetGlobalKey(basePrefix, globalName)

		jsonData, err := json.Marshal(globalVal)
		if err != nil {
			return fmt.Errorf("error marshaling global config '%s': %v", globalKey, err)
		}

		ops = append(ops, clientv3.OpPut(globalKey, string(jsonData)))
	}

	for groupName, groupConfig := range config.Groups {
		groupKey := r.keys.GetGroupKey(basePrefix, groupName)

		jsonData, err := json.Marshal(groupConfig)
		if err != nil {
			return fmt.Errorf("error marshaling group config '%s': %v", groupName, err)
		}

		ops = append(ops, clientv3.OpPut(groupKey, string(jsonData)))
	}

	txnResp, err := r.Client.Txn(ctx).Then(ops...).Commit()
	if err != nil {
		return fmt.Errorf("error executing etcd transaction on storing config: %v", err)
	}
	if !txnResp.Succeeded {
		return fmt.Errorf("etcd transaction failed")
	}

	return nil
}

// DeleteConfig deletes all configuration keys for a specific repository and environment.
func (r *EtcdRepository) DeleteConfig(ctx context.Context, repo, env string) error {
	prefix := r.keys.GetBaseKey(repo, env)
	_, err := r.Client.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("error deleting configuration for repository '%s' and environment '%s': %v", repo, env, err)
	}
	return nil
}

func (r *EtcdRepository) lockEnv(ctx context.Context, env string) (clientv3.LeaseID, error) {
	lease, err := r.Client.Grant(ctx, 5)
	if err != nil {
		return 0, fmt.Errorf("error creating lease: %v", err)
	}

	_, err = r.Client.Put(ctx, r.keys.GetEnvLockKey(env), "lock", clientv3.WithLease(lease.ID))
	if err != nil {
		return 0, fmt.Errorf("error acquiring lock: %v", err)
	}

	return lease.ID, nil
}

func (r *EtcdRepository) releaseEnvLock(ctx context.Context, leaseID clientv3.LeaseID) {
	_, err := r.Client.Revoke(ctx, leaseID)
	if err != nil {
		log.Printf("error revoking lease: %v", err)
	}
}

// IsLockHeld checks if the lock is currently held and waits for it to be released, retrying up to 3 times.
func (r *EtcdRepository) isEnvLockHeld(ctx context.Context, env string) error {
	const retryCount = 3
	const retryDelay = 350 * time.Millisecond // TODO

	for i := 0; i < retryCount; i++ {
		res, err := r.Client.Get(ctx, r.keys.GetEnvLockKey(env))
		if err != nil {
			return fmt.Errorf("error checking lock for '%s' environment: %v", env, err)
		}
		if res.Count == 0 {
			return nil
		}
		if i < retryCount-1 {
			time.Sleep(retryDelay)
		}
	}

	return nil
}

func (r *EtcdRepository) ReadConfig(ctx context.Context, repo, env string, groups, globalKeys []string) (map[string]interface{}, error) {
	err := r.isEnvLockHeld(ctx, env)
	if err != nil {
		return nil, fmt.Errorf("error verifying the lock while reading config for environment '%s': %v", env, err)
	}

	result := map[string]interface{}{}
	for _, groupName := range groups {
		groupKey := r.keys.GetReadGroupRepoEnvKey(repo, env, groupName)
		res, err := r.Client.Get(ctx, groupKey)
		if err != nil {
			return nil, fmt.Errorf("error fetching group '%s' config: %v", groupName, err)
		}

		if len(res.Kvs) > 0 {
			val := res.Kvs[0].Value
			var groupConfig types.GroupConfig

			err = json.Unmarshal([]byte(val), &groupConfig)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling group '%s' config: %v", groupName, err)
			}

			// combine local and referenced global configurations for the group (goroutine?)
			combinedGroupConfig := map[string]interface{}{}

			for localKey, localVal := range groupConfig.Local {
				combinedGroupConfig[localKey] = localVal
			}

			// IMP: there could be many global keys (goroutine?)
			for _, key := range groupConfig.Global {
				if _, ok := combinedGroupConfig[key]; !ok {
					// fetch global value if not already fetched

					gKey := r.keys.GetReadGlobalRepoEnvKey(repo, env, key)
					gRes, err := r.Client.Get(ctx, gKey)
					if err != nil {
						return nil, fmt.Errorf("error reading keys by pattern '%s': %v", gKey, err)
					}
					if len(gRes.Kvs) < 1 {
						log.Printf("key '%s' was not found in '%s' environment while getting keys", key, env)
						continue
					}

					var gVal interface{}
					err = json.Unmarshal([]byte(gRes.Kvs[0].Value), &gVal)
					if err != nil {
						return nil, fmt.Errorf("error unmarshalling global '%s' config for group '%s': %v", key, groupName, err)
					}

					combinedGroupConfig[key] = gVal
				}
			}

			result[groupName] = combinedGroupConfig
		}
	}

	for _, key := range globalKeys {
		if _, ok := result[key]; !ok {
			gKey := r.keys.GetReadGlobalRepoEnvKey(repo, env, key)
			gRes, err := r.Client.Get(ctx, gKey)
			if err != nil {
				return nil, fmt.Errorf("error reading keys by pattern '%s': %v", gKey, err)
			}
			if len(gRes.Kvs) < 1 {
				log.Printf("key '%s' was not found in '%s' environment while getting keys", key, env)
				continue
			}

			var gVal interface{}
			err = json.Unmarshal([]byte(gRes.Kvs[0].Value), &gVal)
			if err != nil {
				return nil, fmt.Errorf("error unmarshalling global config '%s': %v", key, err)
			}

			result[key] = gVal
		}
	}

	return result, nil
}

func (r *EtcdRepository) CloneConfig(ctx context.Context, repo, env, newEnv string, updateGlobal map[string]interface{}) error {
	prefix := r.keys.GetBaseKey(repo, env)

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
				log.Printf("failed to execute transaction for storing cloned config: %v", txnErr)
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
		err := r.DeleteConfig(ctx, repo, newEnv)
		log.Printf("error cleaning the cloned '%s' environment from '%s' environment: %v", newEnv, env, err)
		return err
	}

	if len(updateGlobal) > 0 {
		ops := make([]clientv3.Op, 0, len(updateGlobal))
		prefix := r.keys.GetBaseKey(repo, newEnv)

		for updateKey, updateValue := range updateGlobal {
			key := r.keys.GetGlobalKey(prefix, updateKey)

			jsonData, err := json.Marshal(updateValue)
			if err != nil {
				lastErr = err
				log.Printf("error marshaling global config '%s': %v", key, err)
				break
			}

			ops = append(ops, clientv3.OpPut(key, string(jsonData)))
		}

		if lastErr == nil && len(ops) > 0 {
			_, txnErr := r.Client.Txn(ctx).Then(ops...).Commit()
			if txnErr != nil {
				log.Printf("error cleaning the cloned '%s' environment from '%s' environment: %v", newEnv, env, txnErr)
				lastErr = txnErr
			}
		}
	}

	if lastErr != nil {
		err := r.DeleteConfig(ctx, repo, newEnv)
		log.Printf("error cleaning the cloned '%s' environment from '%s' environment: %v", newEnv, env, err)
		return err
	}

	return nil
}

// AddEnv adds metadata for a new environment to the repository.
func (r *EtcdRepository) AddEnv(ctx context.Context, env string, params EnvParams) error {
	if len(env) < 1 {
		return errors.New("environment name cannot be empty")
	}

	jsonData, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("error marshaling env '%s': %v", env, err)
	}

	_, err = r.Client.Put(ctx, r.keys.GetEnvKey(env), string(jsonData))
	if err != nil {
		return fmt.Errorf("error on adding environment metadata: %w", err)
	}
	return nil
}

// DeleteEnv removes metadata for the specified environment from the repository.
func (r *EtcdRepository) DeleteEnv(ctx context.Context, env string) error {
	if len(env) < 1 {
		return errors.New("environment name cannot be empty")
	}

	res, err := r.Client.Delete(ctx, r.keys.GetEnvKey(env))
	if res.Deleted == 0 {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to remove environment metadata: %w", err)
	}

	return nil
}

// GetEnvOriginal retrieves the original value of the specified environment from the repository.
func (r *EtcdRepository) GetEnvOriginal(ctx context.Context, env string) (string, bool, error) {
	if len(env) < 1 {
		return "", false, errors.New("environment name cannot be empty")
	}

	res, err := r.Client.Get(ctx, r.keys.GetEnvKey(env))
	if err != nil {
		return "", false, fmt.Errorf("error on fetching original '%s' environment value: %w", env, err)
	}
	if res.Count == 0 {
		return "", false, nil
	}

	params := EnvParams{}
	err = json.Unmarshal(res.Kvs[0].Value, &params)
	if err != nil {
		return "", false, fmt.Errorf("error on unmarshal '%s' environment value: %w", env, err)
	}

	return params.Original, true, nil
}

// SetEnvVersion sets the version metadata for the specified environment in the repository.
func (r *EtcdRepository) SetEnvVersion(ctx context.Context, env string, version string) error {
	if len(env) < 1 {
		return errors.New("environment name cannot be empty")
	}

	params, err := r.GetEnvParams(ctx, env)
	if err != nil {
		return err
	}

	params.Version = version

	err = r.AddEnv(ctx, env, params)
	if err != nil {
		return err
	}

	return nil
}

// GetEnvParams retrieves the environment metadata for the specified key.
func (r *EtcdRepository) GetEnvParams(ctx context.Context, env string) (EnvParams, error) {
	if len(env) < 1 {
		return EnvParams{}, errors.New("environment name cannot be empty")
	}

	res, err := r.Client.Get(ctx, r.keys.GetEnvKey(env))
	if err != nil {
		return EnvParams{}, fmt.Errorf("failed to get environment metadata: %w", err)
	}
	if res.Count == 0 {
		return EnvParams{}, EnvNotFoundError{Key: env}
	}

	var params EnvParams
	err = json.Unmarshal(res.Kvs[0].Value, &params)
	if err != nil {
		return EnvParams{}, fmt.Errorf("error unmarshalling '%s' environment params: %v", env, err)
	}

	return params, nil
}

// GetAllEnvs retrieves all available environments from the repository.
func (r *EtcdRepository) GetAllEnvs(ctx context.Context) ([]EnvParams, error) {
	res, err := r.Client.Get(ctx, r.keys.GetEnvKey(""), clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("error on fetching all environments metadata: %w", err)
	}

	envs := make([]EnvParams, 0, len(res.Kvs))
	for _, kvs := range res.Kvs {
		envName := r.extractEnvName(string(kvs.Key))
		envParams, err := r.GetEnvParams(ctx, envName)
		if err != nil {
			return nil, err
		}
		envs = append(envs, envParams)
	}

	return envs, nil
}

// extractEnvName extracts the environment name from the key.
func (r *EtcdRepository) extractEnvName(key string) string {
	return strings.TrimPrefix(key, r.keys.GetEnvKey(""))
}
