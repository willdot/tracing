package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/willdot/tracing/traceprovider"
)

func main() {
	fmt.Println("starting data service")

	srv := &dataservice{}

	rabbitClient, err := setupRabbit(srv)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("connected to rabbit")

	err = traceprovider.JaegerTraceProvider("data-service", "collector", "6831")
	if err != nil {
		log.Fatal(err)
	}

	err = rabbitClient.receive()
	if err != nil {
		log.Fatal(err)
	}
}

type dataservice struct {
	mongoClient *mongoClient
}

func (s *dataservice) handleData(data []byte) error {

	var jsonData map[string]interface{}

	err := json.Unmarshal(data, &jsonData)
	if err != nil {
		return fmt.Errorf("failed to decode data: %w", err)
	}

	for k, v := range jsonData {
		fmt.Println(k)
		fmt.Println(v)
	}

	// TODO: now save this in mongo

	return nil
}
