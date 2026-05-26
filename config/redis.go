package config

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var Ctx = context.Background()

func (cfg Config) NewRedisClient() (*redis.Client, error) {
	connect := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
	client := redis.NewClient(&redis.Options{
		Addr:     connect,
		Password: cfg.Redis.Password,
	})

	_, err := client.Ping(Ctx).Result()
	if err != nil {
		log.Error().
			Err(err).
			Str("source", "config.NewRedisClient").
			Msg("Failed to ping redis client")
		return nil, err
	}

	return client, nil
}
