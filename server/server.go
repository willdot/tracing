package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func main() {
	fmt.Println("starting server")

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

	err = jaegerTraceProvider("my-test", "collector", "6831")
	if err != nil {
		log.Fatal(err)
	}

	handler := http.HandlerFunc(serv.Add)
	wrappedHandler := otelhttp.NewHandler(handler, "hello-instrumented")
	http.Handle("/add", wrappedHandler)

	http.HandleFunc("/get", serv.Get)
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err.Error())
	}
}

type server struct {
	redisClient *redis.Client
}

type data struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *server) addToDB(ctx context.Context, d data) error {
	tr := otel.Tracer("http")
	_, span := tr.Start(ctx, "add to db")
	defer span.End()

	res, err := s.redisClient.Set(context.Background(), d.ID, d.Name, time.Minute).Result()
	if err != nil {
		return err
	}

	print(ctx)

	time.Sleep(time.Second)

	fmt.Printf("added to redis: %v\n", res)
	return nil
}

func print(ctx context.Context) {
	tr := otel.Tracer("http")
	_, span := tr.Start(ctx, "print")

	time.Sleep(time.Millisecond * 500)
	defer span.End()

}

func parseData(ctx context.Context, input []byte) (data, error) {
	tr := otel.Tracer("http")
	ctx, span := tr.Start(ctx, "parse-data")
	defer span.End()

	var reqData data
	err := json.Unmarshal(input, &reqData)
	if err != nil {
		return reqData, err
	}

	return reqData, nil
}

func (s *server) Add(w http.ResponseWriter, req *http.Request) {
	tr := otel.Tracer("http")
	ctx, span := tr.Start(req.Context(), "Add")
	defer span.End()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("could not read body: %s\n", err)
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}

	reqData, err := parseData(ctx, body)
	if err != nil {
		fmt.Printf("could not decode body: %s\n", err)
		http.Error(w, "could not decode body", http.StatusBadRequest)
		return
	}

	err = s.addToDB(ctx, reqData)
	if err != nil {
		fmt.Printf("failed to set in redis: %s\n", err)
		http.Error(w, "failed to save request data", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "You added user '%s' with id '%s'", reqData.Name, reqData.ID)
}

func (s *server) Get(w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id param not provided", http.StatusBadRequest)
		return
	}

	res, err := s.redisClient.Get(context.Background(), id).Result()
	if err != nil {
		fmt.Printf("failed to get in redis: %s\n", err)
		http.Error(w, "failed to get ID", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "You asked for ID '%s' which is '%s'\n", id, res)
}
