package main

import (
	"biocad-opcua/data"
	"biocad-opcua/shared"
	"biocad-opcua/web-server/api"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

// Application configuration constants.
const (
	PREFIX = "web-server: "
	LOG    = "/var/log/web/sys.log"
)

var (
	brokerAddress string
	topic         string
)

func parseFlags() {
	flag.StringVar(&brokerAddress, "brokerhost", "", "Address of the message broker")
	flag.StringVar(&topic, "topic", "measures", "Name of the topic to spread measures across the system")

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

	// Create a subscriber.
	sub := shared.NewSubscriber(brokerAddress, topic, logger)
	sub.Connect()
	defer sub.CloseConnection()

	// Create a data controller.
	bounds := map[string]data.Bounds{
		"Temperature": data.Bounds{
			UpperBound: 100,
			LowerBound: 0,
		},
		"Humidity": data.Bounds{
			UpperBound: 86,
			LowerBound: 16,
		},
	}
	measuresController := api.NewMeasuresController(sub, logger, bounds)

	// Assign routing paths.
	router := mux.NewRouter()
	wwwroot := http.FileServer(http.Dir("client"))

	apiRouter := router.PathPrefix("/api").Subrouter()
	measuresController.SetupRoutes(apiRouter)

	router.PathPrefix("/").Handler(wwwroot)
	http.ListenAndServe(":8080", router)
}
