package shared

import (
	"biocad-opcua/data"
	"encoding/json"
	"log"

	"github.com/nats-io/nats.go"
)

// Subscriber listens for new measures from the message broking service.
type Subscriber struct {
	address string
	conn    *nats.Conn
	logger  *log.Logger
	topic   string
	stop    chan interface{}
	fanout  *Fanout
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
				var measure data.Measure
				err = json.Unmarshal(message.Data, &measure)

				go func() {
					subscriber.fanout.SendMeasure(measure)
				}()

			case <-subscriber.stop:
				break
			}
		}
	}()
}

// AddChannelSubscriber adds the given channel to the message broker client's fanout.
func (subscriber *Subscriber) AddChannelSubscriber(channel chan<- data.Measure) {
	subscriber.fanout.AddChannel(channel)
}

// RemoveChannelSubscriber removes the given channel from the message broker client's fanout.
func (subscriber *Subscriber) RemoveChannelSubscriber(channel chan<- data.Measure) error {
	err := subscriber.fanout.RemoveChannel(channel)
	subscriber.handleRemoveSubscriptionError(err)

	return err
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
		address: address,
		topic:   topic,
		logger:  logger,
		fanout:  NewFanout(),
		stop:    make(chan interface{}),
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

func (subscriber *Subscriber) handleRemoveSubscriptionError(err error) {
	if err != nil {
		subscriber.logger.Println("Couldn't remove the channel from the fanout", err)
	}
}
