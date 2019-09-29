package api

import (
	"biocad-opcua/web-server/model"
	"biocad-opcua/web-server/subscriber"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

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
	parameters, ok := r.URL.Query()["parameter"]

	if !ok || len(parameters) < 1 {
		ctl.handleWebError(w, http.StatusBadRequest, "Argument 'parameter' is missing")
		return
	}

	parameter := parameters[0]
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
	parameters, ok := r.URL.Query()["parameter"]

	if !ok || len(parameters) < 1 {
		ctl.handleWebError(w, http.StatusBadRequest, "Argument 'parameter' is missing")
		return
	}

	lowerBounds, ok := r.URL.Query()["lower_bound"]

	if !ok || len(lowerBounds) < 1 {
		ctl.handleWebError(w, http.StatusBadRequest, "Argument 'lower_bound' is missing")
		return
	}

	upperBounds, ok := r.URL.Query()["upper_bound"]

	if !ok || len(upperBounds) < 1 {
		ctl.handleWebError(w, http.StatusBadRequest, "Argument 'upper_bound' is missing")
		return
	}

	parameter := parameters[0]
	lowerBoundStr := lowerBounds[0]
	upperBoundStr := upperBounds[0]

	if _, ok := ctl.bounds["parameter"]; !ok {
		ctl.handleWebError(w, http.StatusNotFound,
			fmt.Sprintf("Parameter '%s' is not monitored on the server", parameter))

		return
	}

	lowerBound, err := strconv.ParseFloat(lowerBoundStr, 64)

	if err != nil {
		ctl.handleWebError(w, http.StatusBadRequest, "Argument 'lower_bound' is incorrect")
		return
	}

	upperBound, err := strconv.ParseFloat(upperBoundStr, 64)

	if err != nil {
		ctl.handleWebError(w, http.StatusBadRequest, "Argument 'upper_bound' is incorrect")
		return
	}

	if upperBound <= lowerBound {
		ctl.handleWebError(w, http.StatusBadRequest, "Argument 'lower_bound' should be less than 'upper_bound'")
		return
	}

	ctl.bounds[parameter] = model.Bounds{
		LowerBound: lowerBound,
		UpperBound: upperBound,
	}
}

// SetupRoutes sets up HTTP routes for the controller.
func (ctl *MeasuresController) SetupRoutes(router *mux.Router) {
	router.Use(jsonMiddleware)

	router.HandleFunc("/measures", ctl.measures)
	router.HandleFunc("/change_bounds", ctl.changeBoundsForParameter).Methods("PATCH")
	router.HandleFunc("/bounds", ctl.getBoundsForParameter).Methods("GET")
}

// NewMeasuresController returns a new measures controller for the monitored parameters.
func NewMeasuresController(sub *subscriber.Subscriber, logger *log.Logger) *MeasuresController {
	ctl := new(MeasuresController)
	ctl.sub = sub
	ctl.logger = logger

	return ctl
}
