package client

import (
	"net/http"
	"time"
)

// Config defines the configuration for the JSON RPC client.
type Config struct {
	url        string
	timeout    time.Duration
	httpClient *http.Client
	header     http.Header
}

// Option is a function type used to configure the JSON RPC client.
type Option func(opts *Config)

// DefaultOption returns a default set of options.
func DefaultOption() Config {
	return Config{
		timeout:    30 * time.Second,
		httpClient: http.DefaultClient,
	}
}

// URLOption sets the URL for the JSON RPC client.
func URLOption(url string) Option {
	return func(opts *Config) {
		opts.url = url
	}
}

// HTTPClientOption sets the HTTP client for the JSON RPC client.
func HTTPClientOption(httpClient *http.Client) Option {
	return func(opts *Config) {
		opts.httpClient = httpClient
	}
}

// HeaderOption sets the header for the JSON RPC client.
func HeaderOption(header http.Header) Option {
	return func(opts *Config) {
		opts.header = header
	}
}
