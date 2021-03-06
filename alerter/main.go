package main

import (
	"biocad-opcua/alerter/alerting"
	"biocad-opcua/data"
	"biocad-opcua/shared"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

// Application configuration constants.
const (
	PREFIX = "alerter: "
	LOG    = "/var/log/alerter/sys.log"
)

var (
	cacheAddress  string
	dbAddress     string
	database      string
	brokerAddress string
	topic         string
	capacity      int
	launchTimeout int
)

func parseFlags() {
	flag.StringVar(&cacheAddress, "cacheaddress", "", "Address and port of the cache server")
	flag.StringVar(&dbAddress, "dbaddress", "http://localhost:8086",
		"Addres of the database server")
	flag.StringVar(&database, "database", "system_indicators", "Name of the database to store data")
	flag.StringVar(&brokerAddress, "brokerhost", "", "Address of the message broker")
	flag.StringVar(&topic, "topic", "measures", "Name of the topic to spread measures across the system")
	flag.IntVar(&capacity, "capacity", 60, "Number of points per measurment series")
	flag.IntVar(&launchTimeout, "launch-timeout", 5, "Time to sleep before starting the application")

	flag.Parse()
}

func main() {
	parseFlags()

	// Sleep to give other microservices time to start up.
	time.Sleep(time.Duration(launchTimeout) * time.Second)

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

	// Create a cache client.
	cache := shared.NewCache(cacheAddress, logger)
	cache.Connect()
	defer cache.CloseConnection()

	// Create a time-series database client.
	dbclient := shared.NewDbClient(dbAddress, database, logger, capacity)
	err = dbclient.Connect()
	handleError(logger, "Couldn't connect to the database", err)
	defer dbclient.CloseConnection()
	dbclient.Start()
	dbChannel := dbclient.GetSubscriptionChannel()

	// Create an alerter.
	alerter := alerting.NewAlerter(cache, logger)
	alerter.Start()
	defer alerter.Stop()
	alertChannel := alerter.GetSubscriptionChannel()

	// Create a message broker subscriber.
	subscriber := shared.NewSubscriber(brokerAddress, topic, logger)
	err = subscriber.Connect()
	handleError(logger, "Couldn't connect to the message broker", err)
	defer subscriber.CloseConnection()

	// Check for new parameters and set the default bounds for them.
	parameters, err := cache.GetAllParameters()
	handleError(logger, "Couldn't get a list of parameters from the cache", err)

	for _, parameter := range parameters {
		exists, err := cache.CheckParameterBoundsExist(parameter)
		handleError(logger, "Couldn't check if the parameter exists in the cache", err)

		if !exists {
			err = cache.SetParameterBounds(parameter, data.DefaultBounds())
			handleError(logger, "Couldn't set the default bounds for the parameter", err)
		}
	}

	// Channel subscriptions.
	alerter.AddChannelSubscriber(dbChannel)
	alerter.Start()
	defer alerter.Stop()

	subscriber.AddChannelSubscriber(alertChannel)
	subscriber.Start()
	defer subscriber.Stop()

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
