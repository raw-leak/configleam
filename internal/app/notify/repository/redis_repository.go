package repository

import (
	"context"
	"log"

	rds "github.com/raw-leak/configleam/internal/pkg/redis"
)

type RedisRepository struct {
	*rds.Redis
	keys Keys
}

func NewRedisRepository(redis *rds.Redis) *RedisRepository {
	return &RedisRepository{redis, Keys{}}
}

func (r *RedisRepository) Publish(ctx context.Context, payload string) error {
	err := r.Client.Publish(ctx, r.keys.GetNotifyChannel(), payload).Err()
	if err != nil {
		log.Printf("Error publishing update %v", err)
		return PublishUpdateError{}
	}
	return nil
}

func (r *RedisRepository) Subscribe(ctx context.Context, callback func(payload string)) {
	pubsub := r.Client.Subscribe(ctx, r.keys.GetNotifyChannel())
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		callback(msg.Payload)
	}
}

func (r *RedisRepository) Unsubscribe() {

}
