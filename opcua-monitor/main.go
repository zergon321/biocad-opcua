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
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/influxdata/influxdb1-client"
)

// Configuration constants for the application.
const (
	LOG    = "/var/log/opcua/sys.log"
	PREFIX = "monitor: "
)

var (
	endpoint       string
	parametersPath string
	dbAddress      string
	database       string
	cacheAddress   string
	brokerAddress  string
	topic          string
	capacity       int
)

func parseFlags() {
	flag.StringVar(&endpoint, "endpoint", "opc.tcp://localhost:53530/OPCUA/SimulationServer",
		"Address of the OPC UA server")
	flag.StringVar(&parametersPath, "params", "", "File containing OPC UA NodeIDs of the parameters")
	flag.StringVar(&dbAddress, "dbaddress", "http://localhost:8086",
		"Addres of the database server")
	flag.StringVar(&database, "database", "system_indicators", "Name of the database to store data")
	flag.StringVar(&cacheAddress, "cacheaddress", "", "Address and port of the cache server")
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
	ctx := context.Background()
	interval := 1 * time.Second
	monitor := monitoring.NewOpcuaMonitor(ctx, endpoint, logger, interval)
	err = monitor.Connect()
	defer monitor.CloseConnection()
	handleError(logger, "Couldn't connect to the server", err)

	// Create a cache client to store the alerting thresholds for the parameters.
	cache := shared.NewCache(cacheAddress, logger)
	cache.Connect()
	defer cache.CloseConnection()

	// Create a database client and connect to the database.
	dbclient := shared.NewDbClient(dbAddress, database, logger, capacity)
	dbclient.Connect()
	defer dbclient.CloseConnection()

	// Create a publisher to spread measures across the application.
	pb := shared.NewPublisher(brokerAddress, topic, logger)
	pb.Connect()
	defer pb.CloseConnection()

	// Load parameters (NodeIDs) from the file and monitor them.
	parameters, err := monitoring.LoadParametersFromFile(parametersPath)
	handleError(logger, "Couldn't open file with parameters", err)

	// Monitor all the parameters from the file.
	for _, parameter := range parameters {
		// Get the parameter name.
		tokens := strings.Split(parameter, ";")
		tokens = strings.Split(tokens[1], "=")
		name := tokens[1]

		// Check if the parameter exists in the cache.
		exists, err := cache.CheckParameterExists(name)
		handleError(logger, "Couldn't check the parameter for existence", err)

		// If the parameter doesn't exist in the cache, set its bounds to default.
		if !exists {
			err = cache.AddParameters(name)
			handleError(logger, "Couldn't add the parameter name in the cache", err)
		}

		err = monitor.MonitorParameter(parameter)
		handleError(logger, "Couldn't send parameter to monitoring", err)
	}

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

	// Interrupt.
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Kill, os.Interrupt)

	<-interrupt
	logger.Println("Alerter stopped.")
}

func handleError(logger *log.Logger, message string, err error) {
	if err != nil {
		logger.Fatalf("%s: %s", message, err)
	}
}
