package model

import "time"

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
