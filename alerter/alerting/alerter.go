package alerting

import (
	"biocad-opcua/data"
	"biocad-opcua/shared"
	"log"
)

// Alerter listens for measurments and checks them for crossing the alerting thresholds.
type Alerter struct {
	cache  *shared.Cache
	source chan data.Measurement
	fanout *shared.Fanout
	logger *log.Logger
	stop   chan interface{}
}

// Start starts listening for new measurments to check them for crossing alerting thresholds.
func (alerter *Alerter) Start() {
	go func() {
		for {
			select {
			case measure := <-alerter.source:
				params, ok := measure.(data.ParametersState)

				if !ok {
					alerter.logger.Println("Type assertion failed")

					continue
				}

				go alerter.checkParametersForAlerts(params)

			case <-alerter.stop:
				alerter.logger.Println("Alerter stopped")
				break
			}
		}
	}()
}

// GetSubscriptionChannel returns a channel to receive new measurements.
func (alerter *Alerter) GetSubscriptionChannel() chan<- data.Measurement {
	return alerter.source
}

// AddChannelSubscriber adds a new channel to the alerter's fanout to listen for new alerts.
func (alerter *Alerter) AddChannelSubscriber(channel chan<- data.Measurement) {
	alerter.fanout.AddChannel(channel)
}

// RemoveChannelSubscriber removes the channel from the fanout.
func (alerter *Alerter) RemoveChannelSubscriber(channel chan<- data.Measurement) error {
	err := alerter.fanout.RemoveChannel(channel)
	alerter.handleRemoveSubscriberError(err)

	return err
}

// Stop stops listening for new measurements.
func (alerter *Alerter) Stop() {
	alerter.stop <- true
}

// NewAlerter creates a new alerter to warn about alerting thresholds crossing.
func NewAlerter(cache *shared.Cache, logger *log.Logger) *Alerter {
	return &Alerter{
		cache:  cache,
		source: make(chan data.Measurement),
		fanout: shared.NewFanout(),
		logger: logger,
		stop:   make(chan interface{}),
	}
}

func (alerter *Alerter) checkParametersForAlerts(params data.ParametersState) {
	for parameter, value := range params.Parameters {
		bounds, err := alerter.cache.GetParameterBounds(parameter)

		if err != nil {
			continue
		}

		// Check if the alerting thresholds for the parameter are crossed.
		if value < bounds.LowerBound || value > bounds.UpperBound {
			alert := data.Alert{
				Parameter: parameter,
				Bounds:    bounds,
				Value:     value,
				Timestamp: params.Timestamp,
			}

			alerter.logger.Printf("Alert: Time: %v\nParameter: %s\nLower bound: %f\n"+
				"Upper bound: %f\nValue: %f\n", alert.Timestamp, alert.Parameter, alert.LowerBound,
				alert.UpperBound, alert.Value)

			alerter.fanout.SendMeasurement(alert)
		}
	}
}

func (alerter *Alerter) handleRemoveSubscriberError(err error) {
	if err != nil {
		alerter.logger.Println("Couldn't remove the subscriber:", err)
	}
}
