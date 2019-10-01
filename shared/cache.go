package shared

import (
	"github.com/go-redis/redis"
)

type Cache struct {
}

func (cache *Cache) Connect() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}
