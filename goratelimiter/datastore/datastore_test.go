package datastore_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/metaphi-org/go-rate-limiter/goratelimiter/datastore"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func testWithDatastore(t *testing.T, ds datastore.Datastore) {
	ctx := context.TODO()

	testKey1 := "test_key_1"
	testKey2 := "test_key_2"

	incrValues, errors := ds.IncrKeys(ctx, []datastore.KeyConfig{
		{
			Key:         testKey1,
			MaxLifespan: 3 * time.Second,
		},
	})
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, 1, len(incrValues))

	assert.Nil(t, errors[0], "error while incrementing key", testKey1)
	assert.Equal(t, 1, incrValues[0], "initial incr value should be 1")

	incrValues, errors = ds.IncrKeys(ctx, []datastore.KeyConfig{
		{
			Key:         testKey1,
			MaxLifespan: 3 * time.Second,
		},
		{
			Key:         testKey2,
			MaxLifespan: 3 * time.Second,
		},
	})
	assert.Equal(t, 2, len(errors))
	assert.Equal(t, 2, len(incrValues))

	assert.Nil(t, errors[0], "error while incrementing key", testKey1)
	assert.Nil(t, errors[1], "error while incrementing key", testKey2)

	assert.Equal(t, 2, incrValues[0], "subsequent incr value should be 2")
	assert.Equal(t, 1, incrValues[1], "initial incr value should be 1")

	time.Sleep(15 * time.Second) // high as local dynamodb is not perfect in managing TTLs
	incrValues, errors = ds.IncrKeys(ctx, []datastore.KeyConfig{
		{
			Key:         testKey1,
			MaxLifespan: 3 * time.Second,
		},
		{
			Key:         testKey2,
			MaxLifespan: 3 * time.Second,
		},
	})
	assert.Equal(t, 2, len(errors))
	assert.Equal(t, 2, len(incrValues))

	assert.Nil(t, errors[0], "error while incrementing key", testKey1)
	assert.Nil(t, errors[1], "error while incrementing key", testKey2)

	assert.Equal(t, 1, incrValues[0], "incr value should be 1 after ttl elapsed")
	assert.Equal(t, 1, incrValues[1], "incr value should be 1 after ttl elapsed")
}

func TestRedisDatastore(t *testing.T) {
	ds := datastore.RedisDatastore{
		Client: redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		}),
	}

	testWithDatastore(t, ds)
}

func TestDynamoDbDatastore(t *testing.T) {
	// aws dynamodb create-table --endpoint-url http://localhost:8000 --attribute-definitions AttributeName=pk,AttributeType=S  --table-name rate_limiter_test  --key-schema AttributeName=pk,KeyType=HASH --billing-mode PAY_PER_REQUEST
	// aws dynamodb update-time-to-live --endpoint-url http://localhost:8000 --table-name rate_limiter_test --time-to-live-specification "Enabled=true, AttributeName=ttl"

	os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Panicln("unable to load aws config", err)
	}

	cfg.BaseEndpoint = aws.String("http://localhost:8000")

	testWithDatastore(
		t,
		datastore.NewDynamoDBDatastore(
			cfg,
			"rate_limiter_test",
			func(id string) map[string]string {
				return map[string]string{
					"pk": id,
				}
			},
			"ttl",
			"incr_count",
		),
	)
}
