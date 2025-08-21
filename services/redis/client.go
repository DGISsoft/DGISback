// redis/client.go
package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// RedisClient определяет интерфейс для работы с Redis
type RedisClient interface {
	Set(key string, value interface{}) error
	Get(key string) (string, error)
	Delete(key string) error
	// Publish теперь принимает context.Context
	Publish(ctx context.Context, channel string, message interface{}) error
	// Subscribe теперь принимает context.Context
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
}

// redisClient является конкретной реализацией RedisClient, используя go-redis
type redisClient struct {
	client *redis.Client
}

// Delete удаляет ключ из Redis
func (r *redisClient) Delete(key string) error {
	return r.client.Del(context.Background(), key).Err()
}

// Set устанавливает значение для ключа в Redis
func (r *redisClient) Set(key string, value interface{}) error {
	return r.client.Set(context.Background(), key, value, 0).Err()
}

// Get получает значение по ключу из Redis
func (r *redisClient) Get(key string) (string, error) {
	return r.client.Get(context.Background(), key).Result()
}

// Publish публикует сообщение в указанный канал, используя переданный контекст
func (r *redisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	return r.client.Publish(ctx, channel, message).Err()
}

// Subscribe подписывается на указанные каналы, используя переданный контекст
func (r *redisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}

// NewRedisClient создает новый экземпляр RedisClient
func NewRedisClient(addr string, password string, db int) RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &redisClient{client: rdb}
}