package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel"
)

func (s *server) addAPIKeyToRedis(ctx context.Context, apiKey string) error {
	tr := otel.Tracer("account-server")
	_, span := tr.Start(ctx, "add-api-key-to-redis")
	defer span.End()

	res, err := s.redisClient.Set(ctx, apiKey, 1, time.Minute*5).Result()
	if err != nil {
		return err
	}

	fmt.Printf("added to redis: %v\n", res)
	return nil
}

func (s *server) checkAPIKeyInRedis(ctx context.Context, apiKey string) (bool, error) {
	tr := otel.Tracer("account-server")
	_, span := tr.Start(ctx, "check-api-key-in-redis")
	defer span.End()

	_, err := s.redisClient.Get(ctx, apiKey).Result()

	if err == nil {
		return true, nil
	}

	if err == redis.Nil {
		return false, nil
	}

	return false, err
}
