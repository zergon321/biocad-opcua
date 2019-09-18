package parameters

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
	logger        log.Logger
	interval      time.Duration
	parameters    map[uint32]string
	handleCounter uint32
}

// Connect establishes the connection between the server and the client.
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

// SubscribeToParameter makes the client receive updates
// of the specified parameter from teh server.
func (client *OpcuaClient) SubscribeToParameter(parameter string) error {
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

// Start starts the update receiving loop.
func (client *OpcuaClient) Start() {
	go client.subscription.Run(client.ctx)

	// TODO: receiving parameter updates from the server and sending them to the fan-out.
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
