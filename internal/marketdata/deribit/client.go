package deribit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type Client struct {
	baseURL    string
	wsURL      string
	httpClient *http.Client
	logger     *slog.Logger
	requestID  atomic.Int64
	maxRetries int
}

type Config struct {
	BaseURL    string
	WSURL      string
	Timeout    time.Duration
	MaxRetries int
	Logger     *slog.Logger
}

type VolatilityIndexDataParams struct {
	Currency       string `json:"currency"`
	StartTimestamp int64  `json:"start_timestamp"`
	EndTimestamp   int64  `json:"end_timestamp"`
	Resolution     string `json:"resolution"`
}

type InstrumentQuery struct {
	Currency string `json:"currency"`
	Kind     string `json:"kind,omitempty"`
	Expired  bool   `json:"expired,omitempty"`
}

type OrderBookQuery struct {
	InstrumentName string `json:"instrument_name"`
	Depth          int    `json:"depth,omitempty"`
}

type VolatilityIndexDataResponse struct {
	Data         []VolatilityIndexCandle `json:"data"`
	Continuation *int64                  `json:"continuation"`
}

type VolatilityIndexCandle struct {
	Timestamp int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
}

func (c *VolatilityIndexCandle) UnmarshalJSON(data []byte) error {
	var raw []float64
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw) != 5 {
		return fmt.Errorf("unexpected volatility candle length %d", len(raw))
	}

	c.Timestamp = int64(raw[0])
	c.Open = raw[1]
	c.High = raw[2]
	c.Low = raw[3]
	c.Close = raw[4]
	return nil
}

type Instrument struct {
	InstrumentName string  `json:"instrument_name"`
	Kind           string  `json:"kind"`
	BaseCurrency   string  `json:"base_currency"`
	QuoteCurrency  string  `json:"quote_currency"`
	TickSize       float64 `json:"tick_size"`
	MinTradeAmount float64 `json:"min_trade_amount"`
}

type OrderBook struct {
	InstrumentName string      `json:"instrument_name"`
	Timestamp      int64       `json:"timestamp"`
	BestBidPrice   float64     `json:"best_bid_price"`
	BestAskPrice   float64     `json:"best_ask_price"`
	MarkPrice      float64     `json:"mark_price"`
	IndexPrice     float64     `json:"index_price"`
	State          string      `json:"state"`
	Bids           [][]float64 `json:"bids"`
	Asks           [][]float64 `json:"asks"`
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse[T any] struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      int64     `json:"id"`
	Result  T         `json:"result"`
	Error   *rpcError `json:"error"`
	USIn    int64     `json:"usIn"`
	USOut   int64     `json:"usOut"`
	USDiff  int64     `json:"usDiff"`
	Testnet bool      `json:"testnet"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("deribit rpc error %d: %s", e.Code, e.Message)
}

func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		wsURL:   strings.TrimSpace(cfg.WSURL),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger:     logger,
		maxRetries: maxRetries,
	}
}

// GetVolatilityIndexData follows Deribit's documented market-data JSON-RPC method:
// https://docs.deribit.com/api-reference/market-data/public-get_volatility_index_data
func (c *Client) GetVolatilityIndexData(ctx context.Context, params VolatilityIndexDataParams) (VolatilityIndexDataResponse, error) {
	return doRPC[VolatilityIndexDataResponse](c, ctx, "public/get_volatility_index_data", params)
}

// GetInstruments follows the Deribit market-data instrument discovery method:
// https://docs.deribit.com/api-reference/market-data/public-get_instruments
func (c *Client) GetInstruments(ctx context.Context, params InstrumentQuery) ([]Instrument, error) {
	return doRPC[[]Instrument](c, ctx, "public/get_instruments", params)
}

// GetOrderBook follows the Deribit order-book method:
// https://docs.deribit.com/api-reference/market-data/public-get_order_book
func (c *Client) GetOrderBook(ctx context.Context, params OrderBookQuery) (OrderBook, error) {
	return doRPC[OrderBook](c, ctx, "public/get_order_book", params)
}

func doRPC[T any](c *Client, ctx context.Context, method string, params any) (T, error) {
	var zero T
	if c.baseURL == "" {
		return zero, fmt.Errorf("DERIBIT_BASE_URL is required")
	}

	requestID := c.requestID.Add(1)
	payload := rpcRequest{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return zero, err
	}

	url := c.baseURL + "/" + method
	start := time.Now()

	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return zero, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			c.logger.Warn("deribit request failed", "method", method, "attempt", attempt, "error", err, "ws_url", c.wsURL)
			if attempt == c.maxRetries {
				return zero, err
			}
			if sleepErr := sleepBackoff(ctx, attempt); sleepErr != nil {
				return zero, sleepErr
			}
			continue
		}

		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			return zero, readErr
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			err = fmt.Errorf("deribit returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
			c.logger.Warn("deribit request failed", "method", method, "attempt", attempt, "error", err)
			if attempt == c.maxRetries {
				return zero, err
			}
			if sleepErr := sleepBackoff(ctx, attempt); sleepErr != nil {
				return zero, sleepErr
			}
			continue
		}

		var decoded rpcResponse[T]
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return zero, fmt.Errorf("decode deribit response: %w", err)
		}
		if decoded.Error != nil {
			return zero, decoded.Error
		}

		attrs := []any{
			"method", method,
			"attempt", attempt,
			"latency_ms", time.Since(start).Milliseconds(),
			"deribit_us_diff", decoded.USDiff,
			"deribit_testnet", decoded.Testnet,
		}
		if continuation := continuationValue(decoded.Result); continuation != nil {
			attrs = append(attrs, "continuation", *continuation)
		}
		c.logger.Debug("deribit request succeeded", attrs...)
		return decoded.Result, nil
	}

	return zero, fmt.Errorf("deribit request exhausted retries")
}

func sleepBackoff(ctx context.Context, attempt int) error {
	delay := time.Duration(attempt*attempt) * 200 * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func continuationValue(result any) *int64 {
	switch value := result.(type) {
	case VolatilityIndexDataResponse:
		return value.Continuation
	case *VolatilityIndexDataResponse:
		if value == nil {
			return nil
		}
		return value.Continuation
	default:
		return nil
	}
}
