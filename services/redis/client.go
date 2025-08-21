// redis/client.go
package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisClient interface {
	Set(key string, value interface{}) error
	Get(key string) (string, error)
	Delete(key string) error
	Publish(ctx context.Context, channel string, message interface{}) error 
	Subscribe(channels ...string) *redis.PubSub
}

type redisClient struct {
	client *redis.Client
}

func (r *redisClient) Delete(key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *redisClient) Set(key string, value interface{}) error {
	return r.client.Set(ctx, key, value, 0).Err()
}

func (r *redisClient) Get(key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}


func (r *redisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	return r.client.Publish(ctx, channel, message).Err()
}

func (r *redisClient) Subscribe(channels ...string) *redis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}

func NewRedisClient(addr string, password string, db int) RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &redisClient{client: rdb}
}