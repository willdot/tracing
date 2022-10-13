package main

import (
	"context"
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

	err := traceprovider.JaegerTraceProvider("edge-server", "collector", "6831")
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
}

func checkApiKey(ctx context.Context, apiKey string) (bool, error) {
	tr := otel.Tracer("edge-server")
	ctx, span := tr.Start(ctx, "check-api-key")
	defer span.End()

	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://accountserver:8002/checkAPIKey?apiKey=%s", apiKey), nil)
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
		http.Error(w, "failed to validate API key", http.StatusInternalServerError)
		return
	}

	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Write([]byte("so far so good"))
}
