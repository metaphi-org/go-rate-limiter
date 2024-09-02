package goratelimiter

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/metaphi-org/go-rate-limiter/go-rate-limiter/datastore"
)

type Granularity string

const GranularitySecond Granularity = "SECOND"
const GranularityMinute Granularity = "MINUTE"
const GranularityHour Granularity = "HOUR"
const GranularityDay Granularity = "DAY"
const GranularityWeek Granularity = "WEEK"
const GranularityMonth Granularity = "MONTH"

var expiryMap = map[Granularity]time.Duration{
	GranularitySecond: 1 * time.Second,
	GranularityMinute: 1 * time.Minute,
	GranularityHour:   1 * time.Hour,
	GranularityDay:    24 * time.Hour,
	GranularityWeek:   7 * 24 * time.Hour,
	GranularityMonth:  30 * 24 * time.Hour,
}

type RateLimitConfig struct {
	Name        string
	Identifier  string
	Granularity Granularity
	MaxRequests int
}

func getKey(time time.Time, identifier string, granularity Granularity) datastore.KeyConfig {
	var timeString string
	currentTime := time.UTC()

	switch granularity {
	case GranularitySecond:
		timeString = currentTime.Format("20060102150405")
	case GranularityMinute:
		timeString = currentTime.Format("200601021504")
	case GranularityHour:
		timeString = currentTime.Format("2006010215")
	case GranularityDay:
		timeString = currentTime.Format("20060102")
	case GranularityWeek:
		y, w := currentTime.ISOWeek()
		timeString = fmt.Sprintf("%d_W0%d", y, w)
	case GranularityMonth:
		timeString = currentTime.Format("200601")
	}

	identifierHash := sha256.Sum256([]byte(identifier))

	return datastore.KeyConfig{
		Key:         fmt.Sprintf("ratelimiter:%x:%s:%s", identifierHash, granularity, timeString),
		MaxLifespan: expiryMap[granularity],
	}
}

type ConfigResult struct {
	UsedCount int
	Config    RateLimitConfig
}

func (cr ConfigResult) IsBreached() bool {
	return cr.UsedCount > cr.Config.MaxRequests
}

func IsRateLimitBreached(
	ctx context.Context,
	configs []RateLimitConfig,
	ds datastore.Datastore,
) (bool, []ConfigResult, error) {
	incrKeys := make([]datastore.KeyConfig, len(configs))

	for i, config := range configs {
		incrKeys[i] = getKey(
			time.Now(),
			config.Identifier,
			config.Granularity,
		)
	}

	usedCounts, errs := ds.IncrKeys(ctx, incrKeys)
	var finalArr error
	for i, e := range errs {
		if e != nil {
			log.Println("error", "unable to increment key", configs[i], e)
			if finalArr == nil {
				finalArr = errors.New("failed to process some rate limit configs")
			}
		}
	}

	configResults := make([]ConfigResult, len(configs))
	isBreached := false
	for idx, uc := range usedCounts {
		configResults[idx].Config = configs[idx]
		configResults[idx].UsedCount = uc
		if configResults[idx].IsBreached() {
			isBreached = true
		}
	}

	return isBreached, configResults, finalArr
}
