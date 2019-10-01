package api

import (
	"biocad-opcua/data"
	"biocad-opcua/shared"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		if r.Header.Get("Origin") == "http://localhost:8080" {
			return true
		}

		return false
	},
}

// MeasuresController is responsible for handling HTTP requests
// related to monitored OPC UA parameters.
type MeasuresController struct {
	controller
	sub    *shared.Subscriber
	bounds map[string]data.Bounds
}

// measures is a Websocket handler to send monitoring data to the web client.
func (ctl *MeasuresController) measures(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	ctl.handleWebsocketUpgradeError(err)

	if err != nil {
		return
	}

	ctl.logger.Println("Opened a new websocket connection")
	defer ctl.logger.Println("Websocket connection closed")

	source := make(chan data.Measure)
	ctl.sub.AddChannelSubscriber(source)
	defer ctl.sub.RemoveChannelSubscriber(source)

	for measure := range source {
		err = conn.WriteJSON(measure)
		ctl.handleWebsocketSendMessageError(err)

		if err != nil {
			return
		}
	}
}

// getAllParameters sends a list of the monitored parameters to the client.
func (ctl *MeasuresController) getAllParameters(w http.ResponseWriter, r *http.Request) {
	var parameters []string

	for parameter := range ctl.bounds {
		parameters = append(parameters, parameter)
	}

	data, err := json.MarshalIndent(parameters, "", "    ")

	if err != nil {
		ctl.handleInternalError("Couldn't marshal data to JSON", err)
		ctl.handleWebError(w, http.StatusInternalServerError, "Couldn't marshal data to JSON")

		return
	}

	ctl.sendData(w, data)
}

// getBoundsForParameter sends the parameter alerting bounds to the client.
func (ctl *MeasuresController) getBoundsForParameter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	parameter, ok := vars["parameter"]

	if !ok {
		ctl.handleWebError(w, http.StatusNotFound,
			fmt.Sprint("Parameter is missing", parameter))

		return
	}

	bounds, ok := ctl.bounds[parameter]

	if !ok {
		ctl.handleWebError(w, http.StatusNotFound,
			fmt.Sprintf("Parameter '%s' is not monitored on the server", parameter))

		return
	}

	data, err := json.MarshalIndent(bounds, "", "    ")

	if err != nil {
		ctl.handleInternalError("Couldn't marshal the object to JSON", err)
		ctl.handleWebError(w, http.StatusInternalServerError, "Couldn't build a JSON response")

		return
	}

	ctl.sendData(w, data)
}

// changeBoundsForParameter changes the alert bounds for the specified parameter.
func (ctl *MeasuresController) changeBoundsForParameter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	parameter, ok := vars["parameter"]

	if !ok {
		ctl.handleWebError(w, http.StatusBadRequest,
			fmt.Sprint("Parameter is missing", parameter))

		return
	}

	if _, ok := ctl.bounds[parameter]; !ok {
		ctl.handleWebError(w, http.StatusNotFound,
			fmt.Sprintf("Parameter '%s' is not monitored on the server", parameter))

		return
	}

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		ctl.handleWebError(w, http.StatusBadRequest, "Couldn't read request body")
		return
	}

	ctl.logger.Println("Request sent by the client", string(body))

	var bounds data.Bounds
	err = json.Unmarshal(body, &bounds)

	if err != nil {
		ctl.handleInternalError("Couldn't parse JSON", err)
		ctl.handleWebError(w, http.StatusBadRequest, "Couldn't parse JSON data")

		return
	}

	if bounds.UpperBound <= bounds.LowerBound {
		ctl.handleWebError(w, http.StatusBadRequest,
			"Lower alerting bound must be less than upper alerting bound")

		return
	}

	ctl.bounds[parameter] = bounds
}

// SetupRoutes sets up HTTP routes for the controller.
func (ctl *MeasuresController) SetupRoutes(router *mux.Router) {
	router.Use(jsonMiddleware)

	router.HandleFunc("/measures", ctl.measures)
	router.HandleFunc("/{parameter:[A-Z][a-z]+}/bounds", ctl.changeBoundsForParameter).Methods("PATCH")
	router.HandleFunc("/{parameter:[A-Z][a-z]+}/bounds", ctl.getBoundsForParameter).Methods("GET")
	router.HandleFunc("/parameters", ctl.getAllParameters).Methods("GET")
}

// NewMeasuresController returns a new measures controller for the monitored parameters.
func NewMeasuresController(sub *shared.Subscriber, logger *log.Logger, bounds map[string]data.Bounds) *MeasuresController {
	ctl := new(MeasuresController)
	ctl.sub = sub
	ctl.logger = logger
	ctl.bounds = bounds

	return ctl
}
