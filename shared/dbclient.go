package shared

import (
	"biocad-opcua/data"
	"fmt"
	"log"

	influxdb "github.com/influxdata/influxdb1-client/v2"
)

// DbClient represents a client for the time-series database.
type DbClient struct {
	address        string
	database       string
	logger         *log.Logger
	influxClient   influxdb.Client
	subscription   chan data.Measurement
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

	// Create the database if it doesn't exist on the server.
	query := influxdb.Query{
		Command: "SHOW DATABASES",
	}
	response, err := dbclient.influxClient.Query(query)
	dbclient.handleCheckDatabaseExistsError(err)

	if err != nil {
		return err
	}

	// Get the list of databases.
	databases := response.Results[0].Series[0].Values

	for i := range databases {
		dbName, ok := databases[i][0].(string)

		if !ok {
			err := fmt.Errorf("Couldn't get the database name from the result set")
			dbclient.handleCheckDatabaseExistsError(err)

			return err
		}

		if dbclient.database == dbName {
			dbclient.logger.Printf("Database '%s' already exists on the server", dbclient.database)

			return nil
		}
	}

	query = influxdb.Query{
		Command: "CREATE DATABASE " + dbclient.database,
	}
	_, err = dbclient.influxClient.Query(query)

	if err != nil {
		dbclient.handleCheckDatabaseExistsError(err)

		return err
	}

	dbclient.logger.Printf("Created database '%s'", dbclient.database)

	return nil
}

// CloseConnection closes the database connection.
func (dbclient *DbClient) CloseConnection() {
	dbclient.influxClient.Close()
}

// GetSubscriptionChannel returns the channel to accept measurements and write them to the time-series database.
func (dbclient *DbClient) GetSubscriptionChannel() chan<- data.Measurement {
	return dbclient.subscription
}

// Start starts accepting measurements and writing them to the time-series database.
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
			case measurement := <-dbclient.subscription:
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
				point, err := measurement.ToDataPoint()
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

// Stop stops writing measurements to the database.
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
		subscription:   make(chan data.Measurement),
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

func (dbclient *DbClient) handleCheckDatabaseExistsError(err error) {
	if err != nil {
		dbclient.logger.Println("Couldn't check if the database exists:", err)
	}
}
