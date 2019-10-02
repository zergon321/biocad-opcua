package main

import (
	"biocad-opcua/alerter/alerting"
	"biocad-opcua/shared"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
)

// Application configuration constants.
const (
	PREFIX = "web-server: "
	LOG    = "/var/log/web/sys.log"
)

var (
	cacheAddress  string
	dbAddress     string
	database      string
	brokerAddress string
	topic         string
	capacity      int
)

func parseFlags() {
	flag.StringVar(&cacheAddress, "cacheaddress", "", "Address and port of the cache server")
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

	// Create a cache client.
	cache := shared.NewCache(cacheAddress, logger)
	cache.Connect()
	defer cache.CloseConnection()

	// Create a time-series database client.
	dbclient := shared.NewDbClient(dbAddress, database, logger, capacity)
	dbclient.Connect()
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
	subscriber.Connect()
	defer subscriber.CloseConnection()

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
