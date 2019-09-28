package storage

import (
	"biocad-opcua/opcua-monitor/monitoring"
	"log"
	"strings"

	influxdb "github.com/influxdata/influxdb1-client/v2"
)

func measureToPoint(measure monitoring.Measure) (*influxdb.Point, error) {
	fields := make(map[string]interface{})

	for parameter, value := range measure.Parameters {
		param := strings.ToLower(parameter)
		fields[param] = value
	}

	point, err := influxdb.NewPoint("parameters", nil, fields, measure.Timestamp)

	return point, err
}

// DbClient represents a client for the time-series database.
type DbClient struct {
	address        string
	database       string
	logger         *log.Logger
	influxClient   influxdb.Client
	subscription   chan monitoring.Measure
	pointsInSeries int
	stop           chan interface{}
}

// Connect establishes the connection with the time-series database.
func (dbclient *DbClient) Connect() error {
	client, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr: dbclient.address,
	})
	dbclient.handleDbConnectionError(err)

	if err != nil {
		return err
	}

	dbclient.influxClient = client

	return nil
}

// CloseConnection closes the database connection.
func (dbclient *DbClient) CloseConnection() {
	dbclient.influxClient.Close()
}

// GetSubscriptionChannel returns the channel to accept measures and write them to the time-series database.
func (dbclient *DbClient) GetSubscriptionChannel() chan<- monitoring.Measure {
	return dbclient.subscription
}

// Start starts accepting measures and writing them to the time-series database.
func (dbclient *DbClient) Start() {
	go func() {
		var (
			counter int
			series  influxdb.BatchPoints
			err     error
		)

		// Initialize the first series of points.
		series, err = influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
			Database:  dbclient.database,
			Precision: "us",
		})
		dbclient.handleCreateSeriesError(err)

		if err != nil {
			return
		}

		for {
			select {
			case measure := <-dbclient.subscription:
				// If series is full - write it to the database and create a new one.
				if (counter+1)%dbclient.pointsInSeries == 0 {
					err = dbclient.influxClient.Write(series)
					dbclient.handleWriteToDbError(err)

					series, err = influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
						Database:  dbclient.database,
						Precision: "us",
					})
					dbclient.handleCreateSeriesError(err)

					if err != nil {
						return
					}
				}

				// If the series is not full yet, add a new point to the series.
				point, err := measureToPoint(measure)
				dbclient.handleCreatePointError(err)

				if err != nil {
					continue
				}

				series.AddPoint(point)
				counter++

			case <-dbclient.stop:
				break
			}
		}
	}()
}

// Stop stops writing measures to the database.
func (dbclient *DbClient) Stop() {
	dbclient.stop <- true
}

// NewDbClient creates a new client for the time-series database.
func NewDbClient(address, database string, logger *log.Logger, pointsInSeries int) *DbClient {
	return &DbClient{
		address:        address,
		database:       database,
		logger:         logger,
		pointsInSeries: pointsInSeries,
		subscription:   make(chan monitoring.Measure),
		stop:           make(chan interface{}),
	}
}

func (dbclient *DbClient) handleDbConnectionError(err error) {
	if err != nil {
		dbclient.logger.Println("Couldn't connect to the database:", err)
	}
}

func (dbclient *DbClient) handleWriteToDbError(err error) {
	if err != nil {
		dbclient.logger.Println("Couldn't write data to the database:", err)
	}
}

func (dbclient *DbClient) handleCreateSeriesError(err error) {
	if err != nil {
		dbclient.logger.Println("Couldn't create a point series:", err)
	}
}

func (dbclient *DbClient) handleCreatePointError(err error) {
	if err != nil {
		dbclient.logger.Println("Couldn't create a point for the database:", err)
	}
}
