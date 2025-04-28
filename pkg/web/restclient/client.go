package restclient

import (
	"context"
	"net/http"
	"time"

	"github.com/colibriproject-dev/colibri-sdk-go/pkg/base/logging"
	"github.com/mercari/go-circuitbreaker"
	"github.com/newrelic/go-agent/v3/newrelic"
)

const (
	timeoutDefault           uint   = 1
	restClientTransaction    string = "REST-CLIENT"
	circuitBreakerMsg        string = "[%s] state changed: old [%s] -> new [%s]"
	errServiceNotAvailable   string = "service not available"
	errResponseWithEmptyBody string = "response returned with empty body and %d status code"
)

// RestClient struct with http client and circuit breaker
// name: Represents the name of the REST client.
// baseURL: Holds the base URL of the REST client.
// retries: Specifies the number of retries for the REST client.
// retrySleep: Indicates the time to sleep between retries.
// client: Is a pointer to an http.Client for making HTTP requests.
// cb: Points to a circuitbreaker.CircuitBreaker for managing circuit breaking behavior in the REST client.
type RestClient struct {
	name       string
	baseURL    string
	retries    uint8
	retrySleep uint
	client     *http.Client
	cb         *circuitbreaker.CircuitBreaker
}

// NewRestClient creates a new REST client based on the provided configuration.
//
// config: A pointer to RestClientConfig containing the configuration details for the REST client.
// Returns a pointer to RestClient.
func NewRestClient(config *RestClientConfig) *RestClient {
	if config == nil {
		return nil
	}

	if config.Timeout == 0 {
		config.Timeout = timeoutDefault
	}

	client := &http.Client{Timeout: time.Duration(config.Timeout) * time.Second}
	client.Transport = newrelic.NewRoundTripper(client.Transport)
	return &RestClient{
		name:    config.Name,
		baseURL: config.BaseURL,
		client:  client,
		cb: circuitbreaker.New(
			circuitbreaker.WithOpenTimeout(time.Second*10),
			circuitbreaker.WithTripFunc(circuitbreaker.NewTripFuncConsecutiveFailures(5)),
			circuitbreaker.WithOnStateChangeHookFn(func(oldState, newState circuitbreaker.State) {
				logging.
					Info(context.Background()).
					Msgf(circuitBreakerMsg, config.Name, string(oldState), string(newState))
			}),
		),
	}
}
