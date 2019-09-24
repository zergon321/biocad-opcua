package main

import (
	"biocad-opcua/opcua-monitor/monitoring"
	"biocad-opcua/opcua-monitor/storage"
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
	endpoint  string
	dbaddress string
	database  string
	capacity  int
)

func parseFlags() {
	flag.StringVar(&endpoint, "endpoint", "opc.tcp://localhost:53530/OPCUA/SimulationServer",
		"Address of the OPC UA server")
	flag.StringVar(&dbaddress, "dbaddress", "http://localhost:8086",
		"Addres of the database server")
	flag.StringVar(&database, "database", "system_indicators", "Name of the database to store data")
	flag.IntVar(&capacity, "capacity", 60, "Number of points per measurment series")

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

	// Create a database client and connect to the database.
	dbclient := storage.NewDbClient(dbaddress, database, logger, capacity)
	dbclient.Connect()
	defer dbclient.CloseConnection()

	// Start the monitor.
	err = monitor.Connect()
	handleError(logger, "Couldn't connect to the server", err)
	monitor.Start()
	defer monitor.Stop()

	// Subscribe to certain parameters.
	monitor.MonitorParameter("Humidity")
	monitor.MonitorParameter("Temperature")

	// Console subscriber.
	go func() {
		channel := make(chan monitoring.Measure)
		monitor.AddSubscriber(channel)

		for measure := range channel {
			bytes, err := json.MarshalIndent(measure, "", "    ")
			handleError(logger, "Couldn't marshal the object to JSON", err)

			fmt.Println("alpha", string(bytes))
		}
	}()

	// Database subscriber.
	channel := dbclient.GetSubscriptionChannel()

	monitor.AddSubscriber(channel)
	dbclient.Start()

	time.Sleep(20 * time.Second)
}

func handleError(logger *log.Logger, message string, err error) {
	if err != nil {
		logger.Fatalf("%s: %s", message, err)
	}
}
