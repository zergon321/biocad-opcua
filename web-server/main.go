package main

import (
	"biocad-opcua/web-server/api"
	"biocad-opcua/web-server/subscriber"
	"flag"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

// Application configuration constants.
const (
	PREFIX = "web-server: "
	LOG    = "var/log/web-server/sys.log"
)

var (
	dbAddress     string
	database      string
	brokerAddress string
	topic         string
)

func parseFlags() {
	flag.StringVar(&dbAddress, "dbAddress", "http://localhost:8086",
		"Addres of the database server")
	flag.StringVar(&database, "database", "system_indicators", "Name of the database to store data")
	flag.StringVar(&brokerAddress, "brokerhost", "", "Address of the message broker")
	flag.StringVar(&topic, "topic", "measures", "Name of the topic to spread measures across the system")

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

	// Create a subscriber.
	sub := subscriber.NewSubscriber(brokerAddress, topic, logger)
	sub.Connect()
	defer sub.CloseConnection()

	// Create a data controller.
	measuresController := api.NewMeasuresController(sub, logger)

	// Assign routing paths.
	router := mux.NewRouter()
	wwwroot := http.FileServer(http.Dir("client"))

	apiRouter := router.PathPrefix("/api").Subrouter()
	measuresController.SetupRoutes(apiRouter)

	router.PathPrefix("/").Handler(wwwroot)
	http.ListenAndServe(":8080", router)
}
