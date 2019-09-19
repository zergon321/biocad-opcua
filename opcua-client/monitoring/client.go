package monitoring

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gopcua/opcua/ua"

	"github.com/gopcua/opcua"
)

// OpcuaClient is a class for interaction with OPC UA server.
// You just need to connect to the server and then subscribe to certain parameters.
type OpcuaClient struct {
	endpoint      string
	connection    *opcua.Client
	subscription  *opcua.Subscription
	ctx           context.Context
	logger        *log.Logger
	interval      time.Duration
	parameters    map[uint32]string
	handleCounter uint32
	fanout        *Fanout
	stop          chan interface{}
	stopped       bool
}

// Connect establishes connection between server client.
func (client *OpcuaClient) Connect() error {
	opts := []opcua.Option{
		opcua.SecurityModeString("None"),
		opcua.AuthAnonymous(),
	}

	client.connection = opcua.NewClient(client.endpoint, opts...)
	err := client.connection.Connect(client.ctx)
	client.handleConnectionError(err)

	if err != nil {
		return err
	}

	client.subscription, err = client.connection.Subscribe(&opcua.SubscriptionParameters{
		Interval: client.interval,
	})
	client.handleConnectionError(err)

	if err != nil {
		return err
	}

	return nil
}

// CloseConnection closes the connection and stops
// receiving parameter updates.
func (client *OpcuaClient) CloseConnection() {
	client.connection.Close()
}

// MonitorParameter makes the client receive updates
// of the specified parameter from teh server.
func (client *OpcuaClient) MonitorParameter(parameter string) error {
	id, err := ua.ParseNodeID("ns=5;s=" + parameter)
	client.handleSubscriptionError(err)

	if err != nil {
		return err
	}

	request := opcua.NewMonitoredItemCreateRequestWithDefaults(id,
		ua.AttributeIDValue, client.handleCounter)
	res, err := client.subscription.Monitor(ua.TimestampsToReturnBoth, request)
	client.handleSubscriptionError(err)

	if err != nil {
		return err
	}

	if status := res.Results[0].StatusCode; status != ua.StatusOK {
		err = fmt.Errorf("Bad response status")
		client.handleSubscriptionError(err)

		return err
	}

	client.parameters[client.handleCounter] = parameter
	client.handleCounter++

	return nil
}

// AddSubscriber adds a new subscriber for him to receive measures.
func (client *OpcuaClient) AddSubscriber(channel chan<- Measure) {
	client.fanout.AddChannel(channel)
}

// RemoveSubscriber removes the subscriber for him to stop receiving measures.
func (client *OpcuaClient) RemoveSubscriber(channel chan<- Measure) {
	client.fanout.RemoveChannel(channel)
}

// Start starts the update receiving loop.
func (client *OpcuaClient) Start() {
	if !client.stopped {
		client.logger.Println("Attempt to start already working monitor.")
		return
	}

	go client.subscription.Run(client.ctx)

	go func() {
		for {
			select {
			case <-client.ctx.Done():
				client.logger.Println("Disconnected from the server.")
				return

			case <-client.stop:
				client.logger.Println("Monitor stopped")
				return

			case message := <-client.subscription.Notifs:
				switch mes := message.Value.(type) {
				case *ua.DataChangeNotification:
					client.sendMessageToFanout(mes)

				default:
					client.logger.Println("Unknown message type")
				}
			}
		}
	}()

	client.stopped = false
}

// Stop stops the update receiving loop.
func (client *OpcuaClient) Stop() {
	client.stop <- true
	client.stopped = true
}

func (client *OpcuaClient) sendMessageToFanout(message *ua.DataChangeNotification) {
	for _, item := range message.MonitoredItems {
		measure := Measure{
			Timestamp: time.Now(),
			Parameter: client.parameters[item.ClientHandle],
			Value:     item.Value.Value.Value().(float64),
		}

		client.fanout.SendMeasure(measure)
	}
}

func (client *OpcuaClient) handleConnectionError(err error) {
	if err != nil {
		client.logger.Println("Couldn't connect to the OPC UA server:", err)
	}
}

func (client *OpcuaClient) handleSubscriptionError(err error) {
	if err != nil {
		client.logger.Println("Couldn't subscribe to the parameter:", err)
	}
}
