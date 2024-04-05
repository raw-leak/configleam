package repository

import (
	"context"
	"log"

	"github.com/raw-leak/configleam/internal/pkg/etcd"
)

type EtcdRepository struct {
	*etcd.Etcd
	keys Keys
}

func NewEtcdRepository(etcd *etcd.Etcd) *EtcdRepository {
	return &EtcdRepository{etcd, Keys{}}
}

// Publish updates the value of a key, triggering notifications to watchers.
func (r *EtcdRepository) Publish(ctx context.Context, payload string) error {
	_, err := r.Client.Put(ctx, r.keys.GetNotifyChannel(), payload)
	if err != nil {
		log.Printf("Error publishing update to etcd: %v", err)
		return err
	}
	return nil
}

// Subscribe watches changes on a specific key or prefix.
func (r *EtcdRepository) Subscribe(ctx context.Context, callback func(payload string)) {
	watchChan := r.Client.Watch(ctx, r.keys.GetNotifyChannel())

	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			callback(string(event.Kv.Value))
		}
	}
}

// Unsubscribe might be more complex in an etcd context; it may involve canceling the context passed to the watch.
func (r *EtcdRepository) Unsubscribe() {
	// TODO?
}
