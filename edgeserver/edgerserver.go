package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/willdot/tracing/traceprovider"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

func main() {
	fmt.Println("starting edge server")

	rabbitClient, err := setupRabbit()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("connected to rabbit")

	serv := server{
		rabbit: rabbitClient,
	}

	err = traceprovider.JaegerTraceProvider("edge-server", "collector", "6831")
	if err != nil {
		log.Fatal(err)
	}

	addDataHandler := http.HandlerFunc(serv.AddData)
	http.Handle("/add", otelhttp.NewHandler(addDataHandler, "add-data"))

	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err.Error())
	}
}

type server struct {
	rabbit *rabbitClient
}

func (s *server) AddData(w http.ResponseWriter, req *http.Request) {
	tr := otel.Tracer("edge-server")
	ctx, span := tr.Start(req.Context(), "add-data")
	defer span.End()

	apiKey := req.Header.Get("apiKey")
	if apiKey == "" {
		http.Error(w, "missing apiKey header", http.StatusBadRequest)
		return
	}

	// first check if the API key in the header exists
	valid, err := checkApiKey(ctx, apiKey)
	if err != nil {
		fmt.Printf("failed to validate API key: %s\n", err)
		http.Error(w, "failed to validate API key", http.StatusInternalServerError)
		return
	}

	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("could not read body: %s\n", err)
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}

	var reqData map[string]interface{}
	err = json.Unmarshal(body, &reqData)
	if err != nil {
		fmt.Printf("could not decode body: %s\n", err)
		http.Error(w, "could not decode body", http.StatusBadRequest)
		return
	}

	err = s.rabbit.send(reqData)
	if err != nil {
		fmt.Printf("could publish to rabbit: %s\n", err)
		http.Error(w, "failed to process request", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("processed"))
}

func checkApiKey(ctx context.Context, apiKey string) (bool, error) {
	tr := otel.Tracer("edge-server")
	ctx, span := tr.Start(ctx, "check-api-key")
	defer span.End()

	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	url := os.Getenv("ACCOUNT_SERVER_URL")

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/checkAPIKey?apiKey=%s", url, apiKey), nil)
	if err != nil {
		return false, err
	}
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	// probably something else happened so keep going? maybe?
	return true, nil
}
