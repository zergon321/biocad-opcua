package api

import (
	"biocad-opcua/web-server/model"
	"biocad-opcua/web-server/subscriber"
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
}

// MeasuresController is responsible for handling HTTP requests
// related to monitored OPC UA parameters.
type MeasuresController struct {
	controller
	sub    *subscriber.Subscriber
	bounds map[string]model.Bounds
}

// measures is a Websocket handler to send monitoring data to the web client.
func (ctl *MeasuresController) measures(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	ctl.handleWebsocketUpgradeError(err)

	if err != nil {
		return
	}

	source := ctl.sub.GetChannel()
	ctl.sub.Start()
	defer ctl.sub.Stop()

	for measure := range source {
		err = conn.WriteJSON(measure)
		ctl.handleWebsocketSendMessageError(err)

		if err != nil {
			return
		}
	}
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

	data, err := ioutil.ReadAll(r.Body)

	if err != nil {
		ctl.handleWebError(w, http.StatusBadRequest, "Couldn't read request body")
		return
	}

	var bounds model.Bounds
	err = json.Unmarshal(data, &bounds)

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
}

// NewMeasuresController returns a new measures controller for the monitored parameters.
func NewMeasuresController(sub *subscriber.Subscriber, logger *log.Logger, bounds map[string]model.Bounds) *MeasuresController {
	ctl := new(MeasuresController)
	ctl.sub = sub
	ctl.logger = logger
	ctl.bounds = bounds

	return ctl
}
