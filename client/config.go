package client

import (
	"net/http"
	"time"
)

type Config struct {
	url        string
	timeout    time.Duration
	httpClient *http.Client
	header     http.Header
}

type Option func(opts *Config)

// DefaultOption returns a default set of options.
func DefaultOption() Config {
	return Config{
		timeout:    30 * time.Second,
		httpClient: http.DefaultClient,
	}
}

func URLOption(url string) Option {
	return func(opts *Config) {
		opts.url = url
	}
}

func HTTPClientOption(httpClient *http.Client) Option {
	return func(opts *Config) {
		opts.httpClient = httpClient
	}
}

func HeaderOption(header http.Header) Option {
	return func(opts *Config) {
		opts.header = header
	}
}
