package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"golang.org/x/xerrors"
)

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

type respError struct {
	Code    errorCode       `json:"code"`
	Message string          `json:"message"`
	Meta    json.RawMessage `json:"meta,omitempty"`
}

type errorCode int
type params []interface{}

type Client struct {
	cfg Config
}

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
