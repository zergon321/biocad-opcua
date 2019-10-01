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
		Password: "",
		DB:       0,
	})
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

	err := cache.client.HMSet(parameter, fields).Err()
	cache.handleSetParameterBoundsError(err)

	return err
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
