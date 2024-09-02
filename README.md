# go-rate-limiter

`go-rate-limiter` is a lightweight, high-performance Go library for managing rate limits across various granularities (seconds, minutes, hours, etc.). The library is designed for scalability and ease of integration, making it ideal for distributed systems requiring rate-limiting mechanisms.

## Key Features

- **Granular Rate Limiting**: Supports rate limiting across different time granularities, such as seconds, minutes, hours, days, weeks, and months.
- **Customizable**: Define your own rate-limiting configurations with ease.
- **Safe Increments**: Ensures atomic updates and handles race conditions during increments.
- **DynamoDB Support**: Built-in support for DynamoDB as the datastore for distributed rate limiting.

## Installation

```bash
go get github.com/metaphi-org/go-rate-limiter
```

## Usage

### IsRateLimitBreached Function

The core functionality of `go-rate-limiter` revolves around the `IsRateLimitBreached` function, which checks if the rate limit has been breached for a given set of configurations.

#### Function Signature

```go
func IsRateLimitBreached(
    ctx context.Context,
    configs []RateLimitConfig,
    ds datastore.Datastore,
) (bool, []ConfigResult, error)
```

#### Parameters

- **`ctx`**: The context for controlling request timeouts and cancellations.
- **`configs`**: A list of `RateLimitConfig` objects, each specifying the rate limit configuration, including the identifier, granularity, and maximum allowed requests.
- **`ds`**: An implementation of the `Datastore` interface, such as DynamoDB, which handles key increments and TTL management.

#### Return Values

- **`bool`**: Indicates whether any of the provided rate limits have been breached.
- **`[]ConfigResult`**: Provides detailed results for each rate limit configuration, including the current usage count and the original configuration.
- **`error`**: Returns any errors encountered during the rate limit check.

#### Example

```go
configs := []goratelimiter.RateLimitConfig{
    {
        Name:        "API Requests",
        Identifier:  "user_123",
        Granularity: goratelimiter.GranularityMinute,
        MaxRequests: 100,
    },
}

breached, results, err := goratelimiter.IsRateLimitBreached(context.Background(), configs, dynamoDBDatastore)
if err != nil {
    log.Fatal("Error checking rate limit:", err)
}

if breached {
    fmt.Println("Rate limit exceeded!")
} else {
    fmt.Println("Within rate limit.")
}
```

## Supported Datastores

- **DynamoDB**: Efficiently handles rate limit state management with atomic increments and TTL support.

## Contributing

Contributions are welcome! Please submit pull requests or open issues on GitHub.

## License

`go-rate-limiter` is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.
