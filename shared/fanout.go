package shared

import (
	"biocad-opcua/data"
	"fmt"
)

// Fanout manages the channels that receive the data you send.
type Fanout struct {
	channels []chan<- data.Measure
}

// AddChannel adds a new channel in the fanout.
func (fanout *Fanout) AddChannel(channel chan<- data.Measure) {
	fanout.channels = append(fanout.channels, channel)
}

// RemoveChannel removes the channel from the fanout.
func (fanout *Fanout) RemoveChannel(channel chan<- data.Measure) error {
	var (
		index int
		found bool
	)

	// Find the index of the channel.
	for i := range fanout.channels {
		if fanout.channels[i] == channel {
			found = true
			index = i

			break
		}
	}

	if !found {
		return fmt.Errorf("The channel not found")
	}

	fanout.channels = append(fanout.channels[:index], fanout.channels[index+1:]...)

	return nil
}

// SendMeasure sends a new measure to all the channels of the fanout.
func (fanout *Fanout) SendMeasure(measure data.Measure) {
	for _, channel := range fanout.channels {
		go func(ch chan<- data.Measure) {
			ch <- measure
		}(channel)
	}
}

// NewFanout creates a new fanout to serve data to registered channels.
func NewFanout() *Fanout {
	return &Fanout{
		channels: make([]chan<- data.Measure, 0),
	}
}
