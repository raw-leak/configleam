package repository

import (
	"context"

	"github.com/raw-leak/configleam/internal/app/configleam/types"
	"github.com/raw-leak/configleam/internal/pkg/etcd"
)

type ConfigRepository struct {
	etcd *etcd.Etcd
}

func New(etcd *etcd.Etcd) *ConfigRepository {
	return &ConfigRepository{etcd}
}

func (r *ConfigRepository) StoreConfig(ctx context.Context, config *types.ParsedRepoConfig) error {
	// Start a transaction
	// txn := r.etcd.Client.Txn(ctx)

	// Process and store global configurations
	// for key, value := range config.Globals {
	// 	val, err := json.Marshal(value)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	r.etcd.Client.
	// 		txn = txn.Then(r.etcd.Client.OpPut("/config/global/"+key, string(val)))
	// }

	// TODO
	return nil
}

func (r *ConfigRepository) ReadConfig(_ context.Context, groups []string, globalKeys []string) (map[string]interface{}, error) {
	// TODO
	return nil, nil
}
