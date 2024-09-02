package datastore

import (
	"context"
	"time"
)

type KeyConfig struct {
	Key         string
	MaxLifespan time.Duration
}
type Datastore interface {
	// Responsible for incrementing keys. Returns the post increment counts.
	//
	// Should also add relevant expiry based on MaxLifespan of keys to ensure cleanup. Ideally the expiry should be set on the first increment of the key.
	IncrKeys(ctx context.Context, keys []KeyConfig) ([]int, []error)
}
