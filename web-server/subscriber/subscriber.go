package subscriber

import (
	"biocad-opcua/opcua-monitor/monitoring"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

// Subscriber listens for new measures from the message broking service.
type Subscriber struct {
	address  string
	conn     *nats.Conn
	logger   *log.Logger
	receiver chan monitoring.Measure
	topic    string
	stop     chan interface{}
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
		defer sub.Unsubscribe()

		// Get messages, deserialize them and send to listeners.
		for {
			select {
			case message := <-source:
				var measure monitoring.Measure
				err = json.Unmarshal(message.Data, &measure)

				go func() {
					subscriber.receiver <- measure
				}()

			case <-subscriber.stop:
				break
			}
		}
	}()
}

// Stop stops listening for incoming messages.
func (subscriber *Subscriber) Stop() {
	subscriber.stop <- true
}

// CloseConnection closes the connection with the message broker.
func (subscriber *Subscriber) CloseConnection() {
	subscriber.conn.Close()
}

// NewSubscriber create a new subscriber to receive messages from publishers.
func NewSubscriber(address, topic string, logger *log.Logger) *Subscriber {
	return &Subscriber{
		address:  address,
		topic:    topic,
		logger:   logger,
		receiver: make(chan monitoring.Measure),
		stop:     make(chan interface{}),
	}
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

func (subscriber *Subscriber) handleJSONUnmarshalError(err error) {
	if err != nil {
		subscriber.logger.Println("Couldn't deserialize JSON:", err)
	}
}
