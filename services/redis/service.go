// redis/redis.go
package redis

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var (
	// Service - глобальный экземпляр RedisService
	Service *RedisService
	// Глобальный контекст больше не используется напрямую в Publish/Subscribe
	// ctx = context.Background() - УДАЛЕН
)

// RedisService предоставляет методы для работы с Redis через RedisClient
type RedisService struct {
	redisClient RedisClient
}

// Init инициализирует RedisService
func Init(host string, password string, db int) {
	client := NewRedisClient(host, password, db)
	Service = NewRedisService(client)

	if Service == nil {
		log.Fatal("Ошибка подключения к Redis")
	} else {
		log.Println("Redis подключен")
	}
}

// NewRedisService создает новый экземпляр RedisService
func NewRedisService(redisClient RedisClient) *RedisService {
	return &RedisService{redisClient: redisClient}
}

// SetValue устанавливает значение для ключа
func (s *RedisService) SetValue(key string, value interface{}) error {
	err := s.redisClient.Set(key, value)
	if err != nil {
		log.Printf("Redis have error when set a value %v", err)
		return err
	}
	return nil
}

// GetValue получает значение по ключу
func (s *RedisService) GetValue(key string) (string, error) {
	value, err := s.redisClient.Get(key)
	if err == redis.Nil {
		return "", err
	}
	return value, nil
}

// DeleteValue удаляет значение по ключу
func (s *RedisService) DeleteValue(key string) error {
	err := s.redisClient.Delete(key)
	if err != nil {
		log.Printf("Redis have error when delete a value: %v", err)
		return err
	}
	return nil
}

// Publish публикует сообщение в канал, принимая context для правильного управления
func (s *RedisService) Publish(ctx context.Context, channel string, message interface{}) error {
	// Передаем контекст из вызывающего кода
	return s.redisClient.Publish(ctx, channel, message)
}

// Subscribe подписывается на канал, принимая context для правильного управления
func (s *RedisService) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	// Передаем контекст из вызывающего кода
	return s.redisClient.Subscribe(ctx, channel)
}