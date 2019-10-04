package monitoring

import (
	"biocad-opcua/data"
	"biocad-opcua/shared"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gopcua/opcua/ua"

	"github.com/gopcua/opcua"
)

var (
	synchronizer *sync.Mutex
)

func init() {
	synchronizer = new(sync.Mutex)
}

// OpcuaMonitor is a class for interaction with OPC UA server.
// You just need to connect to the server and then subscribe to certain parameters.
type OpcuaMonitor struct {
	endpoint      string
	connection    *opcua.Client
	subscription  *opcua.Subscription
	ctx           context.Context
	logger        *log.Logger
	interval      time.Duration
	parameters    map[uint32]string
	handleCounter uint32
	fanout        *shared.Fanout
	stop          chan interface{}
	stopped       bool
}

// Connect establishes the connection between the server and the monitor.
func (monitor *OpcuaMonitor) Connect() error {
	opts := []opcua.Option{
		opcua.SecurityModeString("None"),
		opcua.AuthAnonymous(),
	}

	monitor.connection = opcua.NewClient(monitor.endpoint, opts...)
	err := monitor.connection.Connect(monitor.ctx)
	monitor.handleConnectionError(err)

	if err != nil {
		return err
	}

	monitor.subscription, err = monitor.connection.Subscribe(&opcua.SubscriptionParameters{
		Interval: monitor.interval,
	})
	monitor.handleConnectionError(err)

	if err != nil {
		return err
	}

	return nil
}

// CloseConnection closes the connection and stops
// receiving parameter updates.
func (monitor *OpcuaMonitor) CloseConnection() {
	monitor.connection.Close()
}

// MonitorParameter makes the monitor receive updates
// of the specified parameter from the server.
func (monitor *OpcuaMonitor) MonitorParameter(parameter string) error {
	// If not subscribed yet.
	if monitor.subscription == nil {
		subscription, err := monitor.connection.Subscribe(&opcua.SubscriptionParameters{
			Interval: monitor.interval,
		})
		monitor.handleConnectionError(err)

		if err != nil {
			return err
		}

		monitor.subscription = subscription
	}

	// Parse NodeID.
	id, err := ua.ParseNodeID(parameter)
	monitor.handleSubscriptionError(err)

	if err != nil {
		return err
	}

	synchronizer.Lock()
	defer synchronizer.Unlock()

	// Subscribe to the parameter.
	request := opcua.NewMonitoredItemCreateRequestWithDefaults(id,
		ua.AttributeIDValue, monitor.handleCounter)
	res, err := monitor.subscription.Monitor(ua.TimestampsToReturnBoth, request)
	monitor.handleSubscriptionError(err)

	if err != nil {
		return err
	}

	if status := res.Results[0].StatusCode; status != ua.StatusOK {
		err = fmt.Errorf("Bad response status")
		monitor.handleSubscriptionError(err)

		return err
	}

	monitor.parameters[monitor.handleCounter] = parameter
	monitor.handleCounter++

	return nil
}

// AddSubscriber adds a new subscriber for him to receive parameters.
func (monitor *OpcuaMonitor) AddSubscriber(channel chan<- data.Measurement) {
	monitor.fanout.AddChannel(channel)
}

// RemoveSubscriber removes the subscriber for him to stop receiving parameters.
func (monitor *OpcuaMonitor) RemoveSubscriber(channel chan<- data.Measurement) error {
	err := monitor.fanout.RemoveChannel(channel)
	monitor.handleRemoveSubscriptionError(err)

	return err
}

// Start starts the update receiving loop.
func (monitor *OpcuaMonitor) Start() {
	if !monitor.stopped {
		monitor.logger.Println("Attempt to start already working monitor.")
		return
	}

	go monitor.subscription.Run(monitor.ctx)

	go func() {
		for {
			select {
			case <-monitor.ctx.Done():
				monitor.logger.Println("Disconnected from the server.")
				return

			case <-monitor.stop:
				monitor.logger.Println("Monitor stopped")
				return

			case message := <-monitor.subscription.Notifs:
				switch mes := message.Value.(type) {
				case *ua.DataChangeNotification:
					monitor.sendParametersToFanout(mes)

				default:
					monitor.logger.Println("Unknown message type")
				}
			}
		}
	}()

	monitor.stopped = false
}

// Stop stops the update receiving loop.
func (monitor *OpcuaMonitor) Stop() {
	monitor.stop <- true
	monitor.stopped = true
}

func (monitor *OpcuaMonitor) sendParametersToFanout(message *ua.DataChangeNotification) {
	measure := data.ParametersState{
		Timestamp:  time.Now(),
		Parameters: make(map[string]float64),
	}

	// Get the values of the monitored parameters.
	for _, item := range message.MonitoredItems {
		parameter := monitor.parameters[item.ClientHandle]
		value := item.Value.Value.Value().(float64)

		measure.Parameters[parameter] = value
	}

	monitor.fanout.SendMeasurement(measure)
}

func (monitor *OpcuaMonitor) handleConnectionError(err error) {
	if err != nil {
		monitor.logger.Println("Couldn't connect to the OPC UA server:", err)
	}
}

func (monitor *OpcuaMonitor) handleSubscriptionError(err error) {
	if err != nil {
		monitor.logger.Println("Couldn't subscribe to the parameter:", err)
	}
}

func (monitor *OpcuaMonitor) handleRemoveSubscriptionError(err error) {
	if err != nil {
		monitor.logger.Println("Couldn't remove subscription from the fanout:", err)
	}
}

// NewOpcuaMonitor creates a new monitor to track data changes on the OPC UA server and translate them to subscribers.
func NewOpcuaMonitor(ctx context.Context, endpoint string, logger *log.Logger, interval time.Duration) *OpcuaMonitor {
	return &OpcuaMonitor{
		endpoint:      endpoint,
		ctx:           ctx,
		logger:        logger,
		interval:      interval,
		parameters:    make(map[uint32]string),
		fanout:        shared.NewFanout(),
		handleCounter: 0,
		stop:          make(chan interface{}),
		stopped:       true,
	}
}
