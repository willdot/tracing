package main

import (
	"encoding/json"
	"os"

	"github.com/streadway/amqp"
)

type rabbitClient struct {
	conn  *amqp.Connection
	ch    *amqp.Channel
	queue amqp.Queue
}

func (c *rabbitClient) shutdown() {
	c.ch.Close()

	c.conn.Close()
}

func (c *rabbitClient) send(data map[string]interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = c.ch.Publish(
		"",           // exchange
		c.queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonData,
		})
	return err
}

func setupRabbit() (*rabbitClient, error) {
	amqpServerURL := os.Getenv("AMQP_SERVER_URL")
	conn, err := amqp.Dial(amqpServerURL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	queue, err := ch.QueueDeclare(
		"data", // name
		false,  // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)

	if err != nil {
		return nil, err
	}

	return &rabbitClient{
		conn:  conn,
		ch:    ch,
		queue: queue,
	}, nil
}
