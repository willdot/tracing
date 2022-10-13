package main

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
)

func (s *server) addToRedis(ctx context.Context, account Account) error {
	tr := otel.Tracer("account_server")
	_, span := tr.Start(ctx, "add to redis")
	defer span.End()

	res, err := s.redisClient.Set(ctx, account.ID, account.Name, time.Minute).Result()
	if err != nil {
		return err
	}

	fmt.Printf("added to redis: %v\n", res)
	return nil
}

func (s *server) getFromRedis(ctx context.Context, id string) (Account, error) {
	tr := otel.Tracer("account_server")
	_, span := tr.Start(ctx, "get from redis")
	defer span.End()

	res, err := s.redisClient.Get(ctx, id).Result()
	if err != nil {
		return Account{}, err
	}

	account := Account{
		ID:   id,
		Name: res,
	}

	return account, nil
}
