package redis

import (
	"log"
	"os"

	"github.com/go-redis/redis/v7"
)

type Service interface {
	GetClient() *redis.Client
}

type service struct {
	redis *redis.Client
}

var (
	redisHost     = os.Getenv("REDIS_HOST")
	redisPassword = os.Getenv("REDIS_PASSWORD")
	redisInstance *service
)

func New() Service {
	// Reuse Connection
	if redisInstance != nil {
		return redisInstance
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: redisPassword,
		DB:       0,
		// DialTimeout:        10 * time.Second,
		// ReadTimeout:        30 * time.Second,
		// WriteTimeout:       30 * time.Second,
		// PoolSize:           10,
		// PoolTimeout:        30 * time.Second,
		// IdleTimeout:        500 * time.Millisecond,
		// IdleCheckFrequency: 500 * time.Millisecond,
		// TLSConfig: &tls.Config{
		// 	InsecureSkipVerify: true,
		// },
	})

	_, err := redisClient.Ping().Result()
	if err != nil {
		log.Fatalf("connect redis error %v", err)
	}

	redisInstance = &service{
		redis: redisClient,
	}

	return redisInstance
}

func (s *service) GetClient() *redis.Client {
	return s.redis
}
