package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const useage = `useage: 
adding a new record: add <id> <name>
getting a record: get <id>`

func main() {
	args := os.Args

	if len(args) <= 1 {
		log.Fatal(useage)
	}
	switch args[1] {
	case "add":
		id := args[2]
		name := args[3]
		add(id, name)
	case "get":
		id := args[2]
		get(id)
	default:
		log.Fatal(useage)
	}
}

func add(id, name string) {
	reqJSON, _ := json.Marshal(map[string]string{
		"name": name,
		"id":   id,
	})

	reqBody := bytes.NewBuffer(reqJSON)

	req, err := http.NewRequest("POST", "http://localhost:8000/add", reqBody)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(string(b))
}

func get(id string) {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:8000/get?id=%s", id), nil)
	if err != nil {
		log.Fatalln(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(string(b))
}
