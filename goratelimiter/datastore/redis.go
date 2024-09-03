package datastore

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

// RedisDatastore represents a datastore backed by Redis.
type RedisDatastore struct {
	Client        *redis.Client
	ClusterClient *redis.ClusterClient
}

// IncrKeys increments the keys in Redis and sets an expiration time on the first increment.
func (r RedisDatastore) IncrKeys(ctx context.Context, keys []KeyConfig) ([]int, []error) {
	incrementCounts := make([]int, len(keys))
	errs := make([]error, len(keys))

	var pipe redis.Pipeliner
	if r.Client != nil {
		pipe = r.Client.TxPipeline()
	} else if r.ClusterClient != nil {
		pipe = r.ClusterClient.TxPipeline()
	}

	if pipe == nil {
		log.Panicln("redis client not specified")
	}

	incrCmds := make([]*redis.IntCmd, len(keys))
	expireCmds := make([]*redis.BoolCmd, len(keys))

	for i, keyConfig := range keys {

		// Increment the key
		incrCmds[i] = pipe.Incr(ctx, keyConfig.Key)

		// Set the Expire only if it's not already set
		expireCmds[i] = pipe.ExpireNX(ctx, keyConfig.Key, keyConfig.MaxLifespan)
	}

	_, fullerr := pipe.Exec(ctx)

	for i, incrCmd := range incrCmds {
		// Get the incremented value
		incrementCounts[i] = int(incrCmd.Val())

		// Get the error if any in the order increment -> expire -> pipeline
		errs[i] = incrCmd.Err()
		if errs[i] == nil {
			errs[i] = expireCmds[i].Err()
		}
		if errs[i] == nil {
			errs[i] = fullerr
		}
	}

	return incrementCounts, errs
}
