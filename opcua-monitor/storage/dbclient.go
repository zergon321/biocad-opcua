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

type DbClient struct {
	address        string
	database       string
	logger         *log.Logger
	influxClient   influxdb.Client
	subscription   chan monitoring.Measure
	pointsInSeries int
}

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

func (dbclient *DbClient) CloseConnection() {
	dbclient.influxClient.Close()
}

func (dbclient *DbClient) GetSubscriptionChannel() chan<- monitoring.Measure {
	return dbclient.subscription
}

func (dbclient *DbClient) Start() {
	go func() {
		var (
			counter int
			series  influxdb.BatchPoints
			err     error
		)

		for measure := range dbclient.subscription {
			if counter%dbclient.pointsInSeries == 0 {
				if series != nil {
					err = dbclient.influxClient.Write(series)
					dbclient.handleWriteToDbError(err)
				}

				series, err = influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
					Database:  dbclient.database,
					Precision: "us",
				})
				dbclient.handleCreateSeriesError(err)

				if err != nil {
					continue
				}
			}

			point, err := measureToPoint(measure)
			dbclient.handleCreatePointError(err)

			if err != nil {
				continue
			}

			series.AddPoint(point)
			counter++
		}
	}()
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
