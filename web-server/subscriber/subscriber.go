package subscriber

import (
	"biocad-opcua/opcua-monitor/monitoring"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

// Subscriber listens for new measures from the message broking service.
type Subscriber struct {
	address      string
	conn         *nats.Conn
	subscription *nats.Subscription
	logger       *log.Logger
	receiver     chan monitoring.Measure
	topic        string
}

// Connect establishes the connection with the message broker.
func (subscriber *Subscriber) Connect() error {
	conn, err := nats.Connect(subscriber.address)
	subscriber.handleConnectionError(err)

	if err != nil {
		return err
	}

	subscriber.conn = conn

	return err
}

// GetChannel returns a channel to obtain data from the subscriber.
func (subscriber *Subscriber) GetChannel() <-chan monitoring.Measure {
	return subscriber.receiver
}

// Start starts listening for new messages from the topic.
func (subscriber *Subscriber) Start() {
	go func() {
		// Subscribe to the topic.
		source := make(chan *nats.Msg, 64)
		sub, err := subscriber.conn.ChanSubscribe(subscriber.topic, source)
		subscriber.handleSubscriptionError(err)

		if err != nil {
			return
		}
		subscriber.subscription = sub

		// Get messages, deserialize them and send to listeners.
		for message := range source {
			var measure monitoring.Measure
			json.Unmarshal(message.Data, &measure)

			subscriber.receiver <- measure
		}
	}()
}

func (subscriber *Subscriber) handleConnectionError(err error) {
	if err != nil {
		subscriber.logger.Println("Couldn't connect to the NATS message broking service:", err)
	}
}

func (subscriber *Subscriber) handleSubscriptionError(err error) {
	if err != nil {
		subscriber.logger.Println("Couldn't subscribe to the topic:", err)
	}
}
