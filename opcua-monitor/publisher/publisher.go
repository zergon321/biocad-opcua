package publisher

import (
	"biocad-opcua/opcua-monitor/monitoring"
	"encoding/json"
	"log"

	nats "github.com/nats-io/nats.go"
)

type Publisher struct {
	conn   *nats.Conn
	logger *log.Logger
	source chan monitoring.Measure
	topic  string
}

func (publisher *Publisher) Connect() error {
	conn, err := nats.Connect(nats.DefaultURL)
	publisher.handleConnectionError(err)

	if err != nil {
		return err
	}

	publisher.conn = conn

	return err
}

func (publisher *Publisher) GetChannel() chan<- monitoring.Measure {
	return publisher.source
}

func (publisher *Publisher) Start() {
	go func() {
		for measure := range publisher.source {
			data, err := json.MarshalIndent(measure, "", "    ")
			publisher.handleJsonMarshalError(err)

			if err != nil {
				continue
			}

			publisher.conn.Publish(publisher.topic, data)
		}
	}()
}

func NewPublisher()

func (publisher *Publisher) handleConnectionError(err error) {
	if err != nil {
		publisher.logger.Println("Couldn't connect to the NATS message broking service", err)
	}
}

func (publisher *Publisher) handleJsonMarshalError(err error) {
	if err != nil {
		publisher.logger.Println("Couldn't serialize the data yo JSON", err)
	}
}
