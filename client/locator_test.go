package client

import (
	"context"
	"net"
	"testing"
)

func TestCreateCarWithFile1(t *testing.T) {
	udpPacketConn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		t.Fatal("ListenPacket err", err)
	}

	// use http3 client
	httpClient, err := NewHTTP3Client(udpPacketConn, true, "")
	if err != nil {
		t.Fatal("NewHTTP3Client err", err)
	}

	locatorURL := "https://120.79.221.36:5000/rpc/v0"
	apiKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJ1c2VyIl0sIklEIjoiMTA1MjQ0MTYwN0BxcS5jb20iLCJOb2RlSUQiOiIiLCJFeHRlbmQiOiIifQ.Yjoxg9JA7SuikMFL0hHMtOANH1CD2v3JKbpkhSC88XQ"

	locator := NewLocator(locatorURL, nil, HTTPClientOption(httpClient))

	schedulerURL, err := locator.GetSchedulerWithAPIKey(context.Background(), apiKey)
	if err != nil {
		t.Fatal("GetSchedulerWithAPIKey err ", err)
	}

	t.Log("schedulerURL ", schedulerURL)
}
