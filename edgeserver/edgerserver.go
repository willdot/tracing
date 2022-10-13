package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/willdot/tracing/traceprovider"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func main() {
	fmt.Println("starting edge server")

	serv := server{}

	err := traceprovider.JaegerTraceProvider("edge_server", "collector", "6831")
	if err != nil {
		log.Fatal(err)
	}

	addHandler := http.HandlerFunc(serv.AddAccount)
	http.Handle("/addaccount", otelhttp.NewHandler(addHandler, "add account"))

	// getHandler := http.HandlerFunc(serv.Get)
	// http.Handle("/get", otelhttp.NewHandler(getHandler, "get"))

	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err.Error())
	}
}

type server struct {
}

type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// func (s *server) Get(w http.ResponseWriter, req *http.Request) {
// 	tr := otel.Tracer("account_server")
// 	ctx, span := tr.Start(req.Context(), "Get")
// 	defer span.End()

// 	id := req.URL.Query().Get("id")
// 	if id == "" {
// 		http.Error(w, "id param not provided", http.StatusBadRequest)
// 		return
// 	}

// 	account, err := s.getFromRedis(ctx, id)
// 	if err != nil {
// 		fmt.Printf("failed to get account from redis: %s\n", err)
// 		http.Error(w, fmt.Sprintf("failed to get account for ID %s", id), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	jsonResp, err := json.Marshal(account)
// 	if err != nil {
// 		fmt.Printf("failed to get encode account from redis: %s\n", err)
// 		http.Error(w, fmt.Sprintf("failed to get account for ID %s", id), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Write(jsonResp)
// }

func (s *server) AddAccount(w http.ResponseWriter, req *http.Request) {
	tr := otel.Tracer("edge_server")
	ctx, span := tr.Start(req.Context(), "Add account")
	defer span.End()

	req, err := http.NewRequest("POST", "http://accountserver:8002/add", req.Body)
	if err != nil {
		fmt.Printf("failed to create account: %s\n", err)
		http.Error(w, "failed to create account", http.StatusInternalServerError)
		return
	}
	req = req.WithContext(ctx)

	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	_, err = client.Do(req)
	if err != nil {
		fmt.Printf("failed to make http request: %s\n", err)
		http.Error(w, "failed to make http request", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "User added")
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
