package client

import (
	"context"
	"encoding/json"
	"net/http"
)

type Locator interface {
	GetSchedulerWithAPIKey(ctx context.Context, apiKey string) (string, error)
}

var _ Locator = (*locator)(nil)

func NewLocator(url string, header http.Header, opts ...Option) Locator {
	options := []Option{URLOption(url), HeaderOption(header)}
	options = append(options, opts...)

	client := NewClient(options...)

	return locator{client: client}
}

type locator struct {
	client *Client
}

func (l locator) GetSchedulerWithAPIKey(ctx context.Context, apiKey string) (string, error) {
	serializedParams := params{
		apiKey,
	}

	req := request{
		Jsonrpc: "2.0",
		Method:  "titan.GetSchedulerWithAPIKey",
		Params:  serializedParams,
		ID:      1,
	}

	rsp, err := l.client.request(ctx, req)
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(rsp.Result)
	if err != nil {
		return "", err
	}

	var schedulerURL string
	err = json.Unmarshal(b, &schedulerURL)
	if err != nil {
		return "", err
	}

	return schedulerURL, nil
}
