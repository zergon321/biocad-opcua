package publisher

import (
	nats "github.com/nats-io/nats.go"
)

type Publisher struct {
}

func (publisher *Publisher) Connect() error {
	conn, err := nats.Connect()
}
