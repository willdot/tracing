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
)

func main() {
	fmt.Println("starting server")

	rdb := redis.NewClient(&redis.Options{
		DB:   0,
		Addr: ":6379",
	})
	res, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("connected to redis: %s\n", res)

	serv := server{
		redisClient: rdb,
	}

	http.HandleFunc("/add", serv.Add)
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

func (s *server) Add(w http.ResponseWriter, req *http.Request) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("could not read body: %s\n", err)
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}

	var reqData data
	err = json.Unmarshal(body, &reqData)
	if err != nil {
		fmt.Printf("could not decode body: %s\n", err)
		http.Error(w, "could not decode body", http.StatusBadRequest)
		return
	}

	res, err := s.redisClient.Set(context.Background(), reqData.ID, reqData.Name, time.Minute).Result()
	if err != nil {
		fmt.Printf("failed to set in redis: %s\n", err)
		http.Error(w, "failed to save request data", http.StatusInternalServerError)
		return
	}

	fmt.Printf("added to redis: %v\n", res)

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
