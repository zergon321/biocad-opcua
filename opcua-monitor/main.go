package main

import (
	"biocad-opcua/data"
	"biocad-opcua/opcua-monitor/monitoring"
	"biocad-opcua/shared"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/influxdata/influxdb1-client"
)

// Configuration constants for the application.
const (
	LOG    = "/var/log/opcua/sys.log"
	PREFIX = "monitor: "
)

var (
	endpoint      string
	dbAddress     string
	database      string
	brokerAddress string
	topic         string
	capacity      int
)

func parseFlags() {
	flag.StringVar(&endpoint, "endpoint", "opc.tcp://localhost:53530/OPCUA/SimulationServer",
		"Address of the OPC UA server")
	flag.StringVar(&dbAddress, "dbaddress", "http://localhost:8086",
		"Addres of the database server")
	flag.StringVar(&database, "database", "system_indicators", "Name of the database to store data")
	flag.StringVar(&brokerAddress, "brokerhost", "", "Address of the message broker")
	flag.StringVar(&topic, "topic", "measures", "Name of the topic to spread measures across the system")
	flag.IntVar(&capacity, "capacity", 60, "Number of points per measurment series")

	flag.Parse()
}

func main() {
	parseFlags()

	// Change working directory to the application directory.
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))

	if err != nil {
		log.Fatalln("Couldn't get current application folder path:", err)
	}

	err = os.Chdir(dir)

	if err != nil {
		log.Fatalln("Couldn't change directory to bin:", err)
	}

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
	err = monitor.Connect()
	defer monitor.CloseConnection()
	handleError(logger, "Couldn't connect to the server", err)

	// Create a database client and connect to the database.
	dbclient := shared.NewDbClient(dbAddress, database, logger, capacity)
	dbclient.Connect()
	defer dbclient.CloseConnection()

	// Create a publisher to spread measures across the application.
	pb := shared.NewPublisher(brokerAddress, topic, logger)
	pb.Connect()
	defer pb.CloseConnection()

	// Monitor certain parameters.
	monitor.MonitorParameter(data.Humidity)
	monitor.MonitorParameter(data.Temperature)

	// Console subscriber.
	go func() {
		channel := make(chan data.Measurement)
		monitor.AddSubscriber(channel)

		for measure := range channel {
			params, ok := measure.(data.ParametersState)

			if !ok {
				handleError(logger, "Couldn't assert the type", fmt.Errorf("invalid type cast"))
			}

			bytes, err := json.MarshalIndent(params, "", "    ")
			handleError(logger, "Couldn't marshal the object to JSON", err)

			fmt.Println("alpha", string(bytes))
		}
	}()

	// Database subscriber.
	channel := dbclient.GetSubscriptionChannel()

	monitor.AddSubscriber(channel)
	dbclient.Start()
	defer dbclient.Stop()

	// Publisher.
	channel = pb.GetChannel()

	monitor.AddSubscriber(channel)
	pb.Start()
	defer pb.Stop()

	// Start the monitor.
	monitor.Start()
	defer monitor.Stop()

	time.Sleep(20 * time.Second)
}

func handleError(logger *log.Logger, message string, err error) {
	if err != nil {
		logger.Fatalf("%s: %s", message, err)
	}
}
