package shared

import (
	"biocad-opcua/data"
	"fmt"
	"log"
	"strconv"

	"github.com/go-redis/redis"
)

// Cache provides an interface to write and read alert thresholds for the parameters.
type Cache struct {
	client   *redis.Client
	endpoint string
	logger   *log.Logger
}

// Connect establishes a new connection with the cache server.
func (cache *Cache) Connect() {
	cache.client = redis.NewClient(&redis.Options{
		Addr:     cache.endpoint,
		Password: "", // No password.
		DB:       0,  // Default database.
	})
}

// CloseConnection gracefully closes the connection to the cache.
func (cache *Cache) CloseConnection() {
	cache.client.Close()
}

// CheckParameterExists checks if the parameter exists in the cache.
func (cache *Cache) CheckParameterExists(parameter string) (bool, error) {
	result, err := cache.client.Exists(parameter).Result()
	cache.handleCheckParameterExistsError(err)

	if err != nil {
		return false, err
	}

	if result > 0 {
		return true, nil
	}

	return false, nil
}

// AddParameters adds new parameter names to the cache.
func (cache *Cache) AddParameters(parameters ...string) error {
	values := make([]interface{}, len(parameters))

	for i := range parameters {
		values[i] = parameters[i]
	}

	// If the parameter doesn't exist, add it to the set.
	err := cache.client.SAdd("parameters", values...).Err()
	cache.handleAddParameterError(err)

	return err
}

// GetAllParameters returns a list of the all parameters from the cache.
func (cache *Cache) GetAllParameters() ([]string, error) {
	parameters, err := cache.client.SMembers("parameters").Result()
	cache.handleGetAllParametersError(err)

	if err != nil {
		return nil, err
	}

	return parameters, nil
}

// GetParameterBounds returns alert thresholds for the parameter.
func (cache *Cache) GetParameterBounds(parameter string) (data.Bounds, error) {
	bounds := data.Bounds{}
	fields, err := cache.client.HGetAll(parameter).Result()
	cache.handleGetParameterBoundsError(err)

	if err != nil {
		return data.Bounds{}, err
	}

	var (
		lowerBound, upperBound string
		ok                     bool
	)

	// Get the lower bound.
	if lowerBound, ok = fields["lower_bound"]; !ok {
		message := "Parameter 'lower_bound' is missing in the cache"

		cache.logger.Println(message)

		return data.Bounds{}, fmt.Errorf(message)
	}

	temp, err := strconv.ParseFloat(lowerBound, 64)
	cache.handleParameterValueCastError(err)

	if err != nil {
		return data.Bounds{}, err
	}

	bounds.LowerBound = temp

	// Get the upper bound.
	if upperBound, ok = fields["upper_bound"]; !ok {
		message := "Parameter 'upper_bound' is missing in the cache"

		cache.logger.Println(message)

		return data.Bounds{}, fmt.Errorf(message)
	}

	temp, err = strconv.ParseFloat(upperBound, 64)
	cache.handleParameterValueCastError(err)

	if err != nil {
		return data.Bounds{}, err
	}

	bounds.UpperBound = temp

	return bounds, nil
}

// SetParameterBounds assigns new alert thresholds to the parameter.
func (cache *Cache) SetParameterBounds(parameter string, bounds data.Bounds) error {
	fields := map[string]interface{}{
		"lower_bound": bounds.LowerBound,
		"upper_bound": bounds.UpperBound,
	}

	// Change the bounds for the parameter.
	err := cache.client.HMSet(parameter, fields).Err()
	cache.handleSetParameterBoundsError(err)

	if err != nil {
		return err
	}

	// If the parameter doesn't exist, add it to the set.
	return cache.AddParameters(parameter)
}

// NewCache creates a new cache client.
func NewCache(endpoint string, logger *log.Logger) *Cache {
	return &Cache{
		endpoint: endpoint,
		logger:   logger,
	}
}

func (cache *Cache) handleGetParameterBoundsError(err error) {
	if err != nil {
		cache.logger.Println("Couldn't get bounds for the parameter:", err)
	}
}

func (cache *Cache) handleParameterValueCastError(err error) {
	if err != nil {
		cache.logger.Println("The parameter value is incorrect:", err)
	}
}

func (cache *Cache) handleSetParameterBoundsError(err error) {
	if err != nil {
		cache.logger.Println("Couldn't set parameter bounds in the cache:", err)
	}
}

func (cache *Cache) handleGetAllParametersError(err error) {
	if err != nil {
		cache.logger.Println("Couldn't obtain a list of parameters from the cache", err)
	}
}

func (cache *Cache) handleAddParameterError(err error) {
	if err != nil {
		cache.logger.Println("Couldn't add the parameter to the cache", err)
	}
}

func (cache *Cache) handleCheckParameterExistsError(err error) {
	if err != nil {
		cache.logger.Println("Couldn't check if the parameter exists in the cache", err)
	}
}

func (cache *Cache) handleSnapshotSaveError(err error) {
	if err != nil {
		cache.logger.Println("Couldn't save the cache snapshot to the disk", err)
	}
}
