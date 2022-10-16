package main

import (
	"fmt"
	"log"
	"os"

	"github.com/streadway/amqp"
)

type dataHandler interface {
	handleData(data []byte) error
}

type rabbitClient struct {
	conn  *amqp.Connection
	ch    *amqp.Channel
	queue amqp.Queue

	dataHandler dataHandler
}

func (c *rabbitClient) shutdown() {
	c.ch.Close()

	c.conn.Close()
}

func (c *rabbitClient) receive() error {
	msgs, err := c.ch.Consume(
		c.queue.Name, // queue
		"",           // consumer
		true,         // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return err
	}

	for d := range msgs {
		log.Printf("Received a message: %s", d.Body)

		err := c.dataHandler.handleData(d.Body)
		if err != nil {
			fmt.Printf("failed to handle data: %s\n", err)
		}
	}
	return nil
}

func setupRabbit(dataHandler dataHandler) (*rabbitClient, error) {
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
		conn:        conn,
		ch:          ch,
		queue:       queue,
		dataHandler: dataHandler,
	}, nil
}
