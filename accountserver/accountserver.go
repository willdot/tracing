package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/willdot/tracing/traceprovider"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func main() {
	fmt.Println("starting account server")

	rdb := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	res, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("connected to redis: %s\n", res)

	serv := server{
		redisClient: rdb,
	}

	err = traceprovider.JaegerTraceProvider("account_server", "collector", "6831")
	if err != nil {
		log.Fatal(err)
	}

	addHandler := http.HandlerFunc(serv.Add)
	http.Handle("/add", otelhttp.NewHandler(addHandler, "add"))

	getHandler := http.HandlerFunc(serv.Get)
	http.Handle("/get", otelhttp.NewHandler(getHandler, "get"))

	if err := http.ListenAndServe(":8002", nil); err != nil {
		log.Fatal(err.Error())
	}
}

type server struct {
	redisClient *redis.Client
}

type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *server) Get(w http.ResponseWriter, req *http.Request) {
	tr := otel.Tracer("account_server")
	ctx, span := tr.Start(req.Context(), "Get")
	defer span.End()

	id := req.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id param not provided", http.StatusBadRequest)
		return
	}

	account, err := s.getFromRedis(ctx, id)
	if err != nil {
		fmt.Printf("failed to get account from redis: %s\n", err)
		http.Error(w, fmt.Sprintf("failed to get account for ID %s", id), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(account)
	if err != nil {
		fmt.Printf("failed to get encode account from redis: %s\n", err)
		http.Error(w, fmt.Sprintf("failed to get account for ID %s", id), http.StatusInternalServerError)
		return
	}

	w.Write(jsonResp)
}

func (s *server) Add(w http.ResponseWriter, req *http.Request) {
	tr := otel.Tracer("account_server")
	ctx, span := tr.Start(req.Context(), "Add")
	defer span.End()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("could not read body: %s\n", err)
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}

	reqData, err := decodeAccount(ctx, body)
	if err != nil {
		fmt.Printf("could not decode body: %s\n", err)
		http.Error(w, "could not decode body", http.StatusBadRequest)
		return
	}

	err = s.addToRedis(ctx, reqData)
	if err != nil {
		fmt.Printf("failed to add account in redis: %s\n", err)
		http.Error(w, "failed to add account", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "You added user '%s' with id '%s'", reqData.Name, reqData.ID)
}

func decodeAccount(ctx context.Context, input []byte) (Account, error) {
	tr := otel.Tracer("account_server")
	ctx, span := tr.Start(ctx, "decode-account")
	defer span.End()

	var reqData Account
	err := json.Unmarshal(input, &reqData)
	if err != nil {
		return reqData, err
	}

	return reqData, nil
}
