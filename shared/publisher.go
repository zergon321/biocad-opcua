package shared

import (
	"biocad-opcua/data"
	"encoding/json"
	"log"

	nats "github.com/nats-io/nats.go"
)

// Publisher sends all incoming messages to other services through message broker service.
type Publisher struct {
	address string
	conn    *nats.Conn
	logger  *log.Logger
	source  chan data.Measurement
	topic   string
	stop    chan interface{}
}

// Connect establishes the connection with the message broker.
func (publisher *Publisher) Connect() error {
	conn, err := nats.Connect(publisher.address)
	publisher.handleConnectionError(err)

	if err != nil {
		return err
	}

	publisher.conn = conn

	return err
}

// GetChannel returns a channel to send data to other microservices.
func (publisher *Publisher) GetChannel() chan<- data.Measurement {
	return publisher.source
}

// Start starts listening for incoming messages to send them to other services.
func (publisher *Publisher) Start() {
	go func() {
		for {
			select {
			case measure := <-publisher.source:
				params, ok := measure.(data.ParametersState)

				if !ok {
					publisher.logger.Println("Type assertion failed for", measure)
					continue
				}

				data, err := json.MarshalIndent(params, "", "    ")
				publisher.handleJSONMarshalError(err)

				if err != nil {
					continue
				}

				err = publisher.conn.Publish(publisher.topic, data)
				publisher.handlePublishError(err)

			case <-publisher.stop:
				break
			}
		}
	}()
}

// Stop stops publishing measures.
func (publisher *Publisher) Stop() {
	publisher.stop <- true
}

// CloseConnection closes the connection woth the message broker.
func (publisher *Publisher) CloseConnection() {
	publisher.conn.Close()
}

// NewPublisher creates a new publisher to listen for new messages to send them to other services of the application.
func NewPublisher(address, topic string, logger *log.Logger) *Publisher {
	return &Publisher{
		address: address,
		topic:   topic,
		logger:  logger,
		source:  make(chan data.Measurement),
		stop:    make(chan interface{}),
	}
}

func (publisher *Publisher) handleConnectionError(err error) {
	if err != nil {
		publisher.logger.Println("Couldn't connect to the NATS message broking service:", err)
	}
}

func (publisher *Publisher) handleJSONMarshalError(err error) {
	if err != nil {
		publisher.logger.Println("Couldn't serialize the data yo JSON:", err)
	}
}

func (publisher *Publisher) handlePublishError(err error) {
	if err != nil {
		publisher.logger.Println("Couldn't send the message to subscribers:", err)
	}
}
