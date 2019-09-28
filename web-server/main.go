package main

import (
	"biocad-opcua/opcua-monitor/monitoring"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Application configuration constants.
const (
	PREFIX = "web-server: "
	LOG    = "var/log/web-server/sys.log"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}
	logger *log.Logger
	source chan monitoring.Measure
)

func measures(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	handleWebsocketUpgradeError(err)

	if err != nil {
		return
	}

	for measure := range source {
		err = conn.WriteJSON(measure)
		handleWebsocketSendMessageError(err)

		if err != nil {
			return
		}
	}
}

func main() {
	// Create a log file and a logger.
	file, err := os.OpenFile(LOG, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		log.Fatalln("Couldn't open log file:", err)
	}
	defer file.Close()

	stream := io.MultiWriter(os.Stdout, file)
	logger = log.New(stream, PREFIX, log.LstdFlags|log.Lshortfile)

	// Assign routing paths.
	router := mux.NewRouter()

	http.ListenAndServe(":8080", router)
}

func handleWebsocketUpgradeError(err error) {
	if err != nil {
		logger.Println("Couldn't upgrade connection to websocket:", err)
	}
}

func handleWebsocketSendMessageError(err error) {
	if err != nil {
		logger.Println("Couldn't send a message through websocket:", err)
	}
}
