package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"golang.org/x/xerrors"
)

// request defines the structure of a JSON RPC request.
type request struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// Response defines a JSON RPC response from the spec
// http://www.jsonrpc.org/specification#response_object
type response struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	ID      interface{} `json:"id"`
	Error   *respError  `json:"error,omitempty"`
}

// respError defines the structure of an error in the JSON RPC response.
type respError struct {
	Code    errorCode       `json:"code"`
	Message string          `json:"message"`
	Meta    json.RawMessage `json:"meta,omitempty"`
}

type errorCode int
type params []interface{}

// Client is the JSON RPC client.
type Client struct {
	cfg Config
}

// NewClient creates a new JSON RPC client with the provided options.
func NewClient(opts ...Option) *Client {
	cfg := DefaultOption()

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.header == nil {
		cfg.header = http.Header{}
	}

	return &Client{
		cfg: cfg,
	}
}

// request sends a JSON RPC request to the server.
func (c *Client) request(ctx context.Context, data request) (*response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.cfg.url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header = c.cfg.header.Clone()
	req.Header.Set("Content-Type", "application/json")

	if ctx != nil {
		req = req.WithContext(ctx)
	}

	resp, err := c.cfg.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rsp response
	err = json.Unmarshal(body, &rsp)
	if err != nil {
		return nil, err
	}

	if rsp.Error != nil {
		return nil, xerrors.New(rsp.Error.Message)
	}

	return &rsp, nil
}
