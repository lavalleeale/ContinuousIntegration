package lib

import (
	"os"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client

func StartRedisClient() {
	opt, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		panic(err)
	}

	Rdb = redis.NewClient(opt)
}
