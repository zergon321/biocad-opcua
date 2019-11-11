package main

import (
	"biocad-opcua/shared"
	"biocad-opcua/web-server/api"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

// Application configuration constants.
const (
	PREFIX = "web-server: "
	LOG    = "/var/log/web/sys.log"
)

var (
	cacheAddress  string
	brokerAddress string
	topic         string
	launchTimeout int
)

func parseFlags() {
	flag.StringVar(&cacheAddress, "cacheaddress", "", "Address and port of the cache server")
	flag.StringVar(&brokerAddress, "brokerhost", "", "Address of the message broker")
	flag.StringVar(&topic, "topic", "measures", "Name of the topic to spread measures across the system")
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

	// Create and launch a subscriber.
	sub := shared.NewSubscriber(brokerAddress, topic, logger)
	err = sub.Connect()
	handleError(logger, "Couldn't connect to the message broker", err)
	defer sub.CloseConnection()
	sub.Start()
	defer sub.Stop()

	// Create a data controller.
	measuresController := api.NewMeasuresController(sub, logger, cache)

	// Assign routing paths.
	router := mux.NewRouter()
	wwwroot := http.FileServer(http.Dir("client"))

	apiRouter := router.PathPrefix("/api").Subrouter()
	measuresController.SetupRoutes(apiRouter)

	router.PathPrefix("/").Handler(wwwroot)
	http.ListenAndServe(":8080", router)
}

func handleError(logger *log.Logger, message string, err error) {
	if err != nil {
		logger.Fatalf("%s: %s", message, err)
	}
}
