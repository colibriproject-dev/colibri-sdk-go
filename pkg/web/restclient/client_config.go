package restclient

// RestClientConfig contains the configuration for a REST client
// Name: The name of the REST client
// BaseURL: The base URL for the REST client
// Timeout: The timeout for requests in seconds
// Retries: The number of retries for failed requests
// RetrySleepInSeconds: The time to sleep between retries in seconds
// ProxyURL: The URL of the proxy server to use for requests (e.g., "http://proxy.example.com:8080")
type RestClientConfig struct {
	Name                string
	BaseURL             string
	Timeout             uint
	Retries             uint8
	RetrySleepInSeconds uint
	ProxyURL            string
}
