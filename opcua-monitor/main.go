package main

import (
	"biocad-opcua/opcua-monitor/monitoring"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	_ "github.com/influxdata/influxdb1-client"
)

// Configuration constants for the application.
const (
	LOG    = "sys.log"
	PREFIX = "monitor: "
)

var (
	endpoint string
)

func parseFlags() {
	flag.StringVar(&endpoint, "endpoint", "opc.tcp://localhost:53530/OPCUA/SimulationServer",
		"Address of the OPC UA server")

	flag.Parse()
}

func main() {
	parseFlags()

	// Create a log file and a logger.
	file, err := os.OpenFile(LOG, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		log.Fatalln("Couldn't open log file:", err)
	}
	defer file.Close()

	stream := io.MultiWriter(os.Stdout, file)
	logger := log.New(stream, PREFIX, log.LstdFlags|log.Lshortfile)

	// Create a monitor.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	interval := 1 * time.Second
	monitor := monitoring.NewOpcuaMonitor(ctx, endpoint, logger, interval)

	// Start the monitor.
	err = monitor.Connect()
	handleError(logger, "Couldn't connect to the server", err)
	monitor.Start()
	monitor.MonitorParameter("Humidity")
	monitor.MonitorParameter("Temperature")

	// Subscriber "alpha".
	go func() {
		channel := make(chan monitoring.Measure)
		monitor.AddSubscriber(channel)

		for measure := range channel {
			bytes, err := json.MarshalIndent(measure, "", "    ")
			handleError(logger, "Couldn't marshal the object to JSON", err)

			fmt.Println("alpha", string(bytes))
		}
	}()

	// Subscriber "bravo".
	go func() {
		channel := make(chan monitoring.Measure)
		monitor.AddSubscriber(channel)

		for measure := range channel {
			bytes, err := json.MarshalIndent(measure, "", "    ")
			handleError(logger, "Couldn't marshal the object to JSON", err)

			fmt.Println("bravo", string(bytes))
		}
	}()

	time.Sleep(20 * time.Second)
}

func handleError(logger *log.Logger, message string, err error) {
	if err != nil {
		logger.Fatalf("%s: %s", message, err)
	}
}