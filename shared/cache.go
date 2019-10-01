package shared

import (
	"github.com/go-redis/redis"
)

type Cache struct {
	client   *redis.Client
	endpoint string
}

func (cache *Cache) Connect() {
	cache.client = redis.NewClient(&redis.Options{
		Addr:     cache.endpoint,
		Password: "",
		DB:       0,
	})
}
