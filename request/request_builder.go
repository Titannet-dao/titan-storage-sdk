package request

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Builder is an IPFS commands request builder.
type Builder struct {
	namespace string
	opts      map[string]string
	headers   http.Header
	body      io.Reader
	baseApi   string
	client    *http.Client
}

func NewBuilder(c *http.Client, baseApi, namespace string, requestHeader http.Header) *Builder {
	return &Builder{
		namespace: namespace,
		baseApi:   baseApi,
		client:    c,
		headers:   requestHeader,
	}
}

// BodyString sets the request body to the given string.
func (r *Builder) BodyString(body string) *Builder {
	return r.Body(strings.NewReader(body))
}

// BodyBytes sets the request body to the given buffer.
func (r *Builder) BodyBytes(body []byte) *Builder {
	return r.Body(bytes.NewReader(body))
}

// Body sets the request body to the given reader.
func (r *Builder) Body(body io.Reader) *Builder {
	r.body = body
	return r
}

// Option sets the given config.
func (r *Builder) Option(key string, value interface{}) *Builder {
	var s string
	switch v := value.(type) {
	case bool:
		s = strconv.FormatBool(v)
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		// slow case.
		s = fmt.Sprint(value)
	}
	if r.opts == nil {
		r.opts = make(map[string]string, 1)
	}
	r.opts[key] = s
	return r
}

// Header sets the given header.
func (r *Builder) Header(name, value string) *Builder {
	if r.headers == nil {
		r.headers = http.Header{}
	}
	r.headers.Set(name, value)
	return r
}

// Post sends the post request and return the response.
func (r *Builder) Post(ctx context.Context) (*response, error) {
	req := NewRequest(ctx, r.baseApi, r.namespace, r.headers)
	req.Opts = r.opts
	req.Body = r.body
	return req.Send(r.client, http.MethodPost)
}

// Get sends the get request and return the response.
func (r *Builder) Get(ctx context.Context) (*response, error) {
	req := NewRequest(ctx, r.baseApi, r.namespace, r.headers)
	req.Opts = r.opts
	req.Body = r.body
	return req.Send(r.client, http.MethodGet)
}

// Head sends the head request and return the response.
func (r *Builder) Head(ctx context.Context) (http.Header, error) {
	req := NewRequest(ctx, r.baseApi, r.namespace, r.headers)
	req.Opts = r.opts
	resp, err := r.client.Head(req.getURL())
	if err != nil {
		return nil, err
	}
	return resp.Header, nil
}

// Exec sends the request a request and decodes the response.
func (r *Builder) Exec(ctx context.Context, res interface{}) error {
	httpRes, err := r.Post(ctx)
	if err != nil {
		return err
	}

	if res == nil {
		lateErr := httpRes.Close()
		if httpRes.Error != nil {
			return httpRes.Error
		}
		return lateErr
	}

	return httpRes.Decode(res)
}
