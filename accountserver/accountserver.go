package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/willdot/tracing/traceprovider"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func main() {
	fmt.Println("starting account server")

	redisURL := os.Getenv("REDIS_URL")
	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	res, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("connected to redis: %s\n", res)

	serv := server{
		redisClient: rdb,
	}

	err = traceprovider.JaegerTraceProvider("account-server", "collector", "6831")
	if err != nil {
		log.Fatal(err)
	}

	addHandler := http.HandlerFunc(serv.AddAPIKey)
	http.Handle("/addAPIKey", otelhttp.NewHandler(addHandler, "add-api-key"))

	getHandler := http.HandlerFunc(serv.CheckAPIKey)
	http.Handle("/checkAPIKey", otelhttp.NewHandler(getHandler, "check-api-key"))

	if err := http.ListenAndServe(":8002", nil); err != nil {
		log.Fatal(err.Error())
	}
}

type server struct {
	redisClient *redis.Client
}

func (s *server) CheckAPIKey(w http.ResponseWriter, req *http.Request) {
	tr := otel.Tracer("account-server")
	ctx, span := tr.Start(req.Context(), "check-api-key")
	defer span.End()

	apiKey := req.URL.Query().Get("apiKey")
	if apiKey == "" {
		http.Error(w, "apiKey param not provided", http.StatusBadRequest)
		return
	}

	found, err := s.checkAPIKeyInRedis(ctx, apiKey)
	if err != nil {
		fmt.Printf("failed to check API key in redis: %s\n", err)
		http.Error(w, "failed to check API key in redis", http.StatusInternalServerError)
		return
	}

	if found {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func (s *server) AddAPIKey(w http.ResponseWriter, req *http.Request) {
	tr := otel.Tracer("account-server")
	ctx, span := tr.Start(req.Context(), "add-api-key")
	defer span.End()

	apiKey := req.Header.Get("apiKey")
	if apiKey == "" {
		http.Error(w, "missing apiKey header", http.StatusBadRequest)
		return
	}

	err := s.addAPIKeyToRedis(ctx, apiKey)
	if err != nil {
		fmt.Printf("failed to add API key in redis: %s\n", err)
		http.Error(w, "failed to add API key", http.StatusInternalServerError)
		return
	}
}
