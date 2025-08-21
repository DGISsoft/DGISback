// redis/redis.go
package redis

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var (
	Service *RedisService
	ctx     = context.Background()
)

type RedisService struct {
	redisClient RedisClient
}

func Init(host string, password string, db int) {
	client := NewRedisClient(host, password, db)
	Service = NewRedisService(client)

	if Service == nil {
		log.Fatal("Ошибка подключения к Redis")
	} else {
		log.Println("Redis подключен")
	}
}

func NewRedisService(redisClient RedisClient) *RedisService {
	return &RedisService{redisClient: redisClient}
}

func (s *RedisService) SetValue(key string, value interface{}) error {
	err := s.redisClient.Set(key, value)
	if err != nil {
		log.Printf("Redis have error when set a value %v", err)
		return err
	}
	return nil
}

func (s *RedisService) GetValue(key string) (string, error) {
	value, err := s.redisClient.Get(key)
	if err == redis.Nil {
		return "", err
	}
	return value, nil
}

func (s *RedisService) DeleteValue(key string) error {
	err := s.redisClient.Delete(key)
	if err != nil {
		log.Printf("Redis have error when delete a value: %v", err)
		return err
	}
	return nil
}


func (s *RedisService) Publish(channel string, message interface{}) error {
	return s.redisClient.Publish(ctx, channel, message)
}

func (s *RedisService) Subscribe(channel string) *redis.PubSub {
	return s.redisClient.Subscribe(channel)
}