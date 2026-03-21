package deribit

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestGetVolatilityIndexData(t *testing.T) {
	client := NewClient(Config{
		BaseURL:    "https://deribit.test/api/v2",
		Timeout:    time.Second,
		MaxRetries: 1,
	})
	client.httpClient = &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(bytes.NewBufferString(
					`{"jsonrpc":"2.0","id":1,"result":{"data":[[1710000000000,60,61,59,60.5]],"continuation":1774032780000},"usIn":1774092899867323,"usOut":1774092899901486,"usDiff":34163,"testnet":false}`,
				)),
			}, nil
		}),
	}

	resp, err := client.GetVolatilityIndexData(context.Background(), VolatilityIndexDataParams{
		Currency:       "BTC",
		StartTimestamp: 1,
		EndTimestamp:   2,
		Resolution:     "60",
	})
	if err != nil {
		t.Fatalf("GetVolatilityIndexData returned error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("data length = %d", len(resp.Data))
	}
	if resp.Data[0].Close != 60.5 {
		t.Fatalf("close = %v", resp.Data[0].Close)
	}
	if resp.Continuation == nil || *resp.Continuation != 1774032780000 {
		t.Fatalf("continuation = %v", resp.Continuation)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
