package goratelimiter_test

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/metaphi-org/go-rate-limiter/goratelimiter"
	"github.com/metaphi-org/go-rate-limiter/goratelimiter/datastore"
	"github.com/stretchr/testify/assert"
)

func TestIsRateLimitBreached(t *testing.T) {

	now := time.Now()
	secondsRemainingInCurrentMinute := 59 - now.Second()
	if secondsRemainingInCurrentMinute < 9 {
		time.Sleep(9 * time.Second)
	}

	configs := []goratelimiter.RateLimitConfig{
		{
			Name:        "org_level",
			Identifier:  "orgid-xyz",
			MaxRequests: 80,
			Granularity: goratelimiter.GranularityMinute,
		},
		{
			Name:        "user_level",
			Identifier:  "userid-abc",
			MaxRequests: 2,
			Granularity: goratelimiter.GranularitySecond,
		},
	}

	ctx := context.TODO()
	ds := GetDynamoDbDataStore(ctx)

	wg := sync.WaitGroup{}
	for batch := 0; batch < 9; batch++ {
		concCount := 10
		breachResults := make([]bool, concCount)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				isBreached, configStatus, err := goratelimiter.IsRateLimitBreached(
					context.TODO(),
					configs,
					ds,
				)

				assert.Nil(t, err)

				breachResults[i] = isBreached

				for _, c := range configStatus {
					if c.IsBreached() {
						log.Println(i, "breached", c.Config.Name, c.Config.MaxRequests, c.UsedCount)
					}
				}
			}(i)
		}
		wg.Wait()

		breached := 0
		for _, b := range breachResults {
			if b {
				breached++
			}
		}

		if batch < 8 {
			assert.Equal(t, 8, breached)
		} else {
			assert.Equal(t, concCount, breached)
		}

		time.Sleep(1 * time.Second)
	}
}

func GetDynamoDbDataStore(ctx context.Context) datastore.Datastore {
	// aws dynamodb create-table --endpoint-url http://localhost:8000 --attribute-definitions AttributeName=pk,AttributeType=S  --table-name rate_limiter_test  --key-schema AttributeName=pk,KeyType=HASH --billing-mode PAY_PER_REQUEST
	// aws dynamodb update-time-to-live --endpoint-url http://localhost:8000 --table-name rate_limiter_test --time-to-live-specification "Enabled=true, AttributeName=ttl"

	os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Panicln("unable to load aws config", err)
	}

	endp := "http://localhost:8000"
	cfg.BaseEndpoint = &endp

	return datastore.NewDynamoDBDatastore(
		cfg,
		"rate_limiter_test",
		func(id string) map[string]string {
			return map[string]string{
				"pk": id,
			}
		},
		"ttl",
		"incr_count",
	)
}
