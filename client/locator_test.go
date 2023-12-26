package client

import (
	"context"
	"net/http"
	"testing"

	"github.com/quic-go/quic-go/http3"
)

func TestCreateCarWithFile1(t *testing.T) {

	locatorURL := "https://120.79.221.36:5000/rpc/v0"
	apiKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJ1c2VyIl0sIklEIjoiMTA1MjQ0MTYwN0BxcS5jb20iLCJOb2RlSUQiOiIiLCJFeHRlbmQiOiIifQ.Yjoxg9JA7SuikMFL0hHMtOANH1CD2v3JKbpkhSC88XQ"

	httpClient := &http.Client{
		Transport: &http3.RoundTripper{},
	}

	locator := NewLocator(locatorURL, nil, HTTPClientOption(httpClient))

	schedulerURL, err := locator.GetSchedulerWithAPIKey(context.Background(), apiKey)
	if err != nil {
		t.Fatal("GetSchedulerWithAPIKey err ", err)
	}

	t.Log("schedulerURL ", schedulerURL)
}
