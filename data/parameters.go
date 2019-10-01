package data

import (
	"strings"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
)

const (
	// Temperature is a physical quantity expressing hot and cold.
	Temperature = "Temperature"
	// Humidity is the concentration of water vapour present in air.
	Humidity = "Humidity"
	// Pressure is the force applied perpendicular to the surface
	// of an object per unit area over which that force is distributed.
	Pressure = "Pressure"
	// Density of a substance is its mass per unit volume.
	Density = "Density"
	// Volume is the quantity of three-dimensional space
	// enclosed by a closed surface.
	Volume = "Volume"
	// Voltage is the difference in electric potential between two points.
	Voltage = "Voltage"
	// Resistance of an object is a measure of its opposition
	// to the flow of electric current.
	Resistance = "Resistance"
	// Amperage is the voltage to resistance ratio.
	Amperage = "Amperage"
	// Wavelength is the spatial period of a periodic wave—the
	// distance over which the wave's shape repeats.
	Wavelength = "Wavelength"
	// Frequency is the number of occurrences of a repeating event per unit of time.
	Frequency = "Frequency"
)

// Measurement represents an object which can treated as a data portion for a time-series database.
type Measurement interface {
	ToDataPoint() (*influxdb.Point, error)
}

// ParametersState represents the state of the system parameters at a certain moment of time.
type ParametersState struct {
	Timestamp  time.Time
	Parameters map[string]float64
}

// ToDataPoint transforms ParametersState object into time-series data point.
func (params ParametersState) ToDataPoint() (*influxdb.Point, error) {
	fields := make(map[string]interface{})

	for parameter, value := range params.Parameters {
		param := strings.ToLower(parameter)
		fields[param] = value
	}

	point, err := influxdb.NewPoint("parameters", nil, fields, params.Timestamp)

	return point, err
}
