package data

import (
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
)

// Alert represents an alert message for a certain parameter.
type Alert struct {
	Parameter string
	Bounds
	Value     float64
	Timestamp time.Time
}

// Bounds are values the parameter should be within.
type Bounds struct {
	LowerBound float64
	UpperBound float64
}

// ToDataPoint transforms alert object into a time-series data point.
func (alert Alert) ToDataPoint() (*influxdb.Point, error) {
	tags := map[string]string{
		"parameter": alert.Parameter,
	}

	fields := map[string]interface{}{
		"lower_bound": alert.LowerBound,
		"upper_bound": alert.UpperBound,
		"value":       alert.Value,
	}

	point, err := influxdb.NewPoint("alerts", tags, fields, alert.Timestamp)

	if err != nil {
		return nil, err
	}

	return point, nil
}
