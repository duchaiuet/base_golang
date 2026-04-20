package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout = 10 * time.Second
	defaultRecvWindow  = 5000
)

// Side represents order direction.
type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// Kline stores a reduced candle representation required by the strategy.
type Kline struct {
	OpenTime  time.Time
	CloseTime time.Time
	Close     float64
}

// OrderResponse stores selected fields from Binance order create response.
type OrderResponse struct {
	Symbol              string `json:"symbol"`
	OrderID             int64  `json:"orderId"`
	Status              string `json:"status"`
	ClientOrderID       string `json:"clientOrderId"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	TransactTime        int64  `json:"transactTime"`
}

// Client wraps Binance HTTP APIs used by the trading agent.
type Client struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// NewClient creates a Binance client.
func NewClient(baseURL, apiKey, apiSecret string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     strings.TrimSpace(apiKey),
		apiSecret:  strings.TrimSpace(apiSecret),
		httpClient: &http.Client{Timeout: timeout},
	}
}

// GetKlines reads recent candle data for a symbol.
func (c *Client) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]Kline, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	params.Set("interval", interval)
	params.Set("limit", strconv.Itoa(limit))

	body, err := c.request(ctx, http.MethodGet, "/api/v3/klines", params, false)
	if err != nil {
		return nil, err
	}

	var raw [][]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode kline response: %w", err)
	}

	klines := make([]Kline, 0, len(raw))
	for i, entry := range raw {
		if len(entry) < 7 {
			return nil, fmt.Errorf("kline entry %d has insufficient fields", i)
		}

		openMS, err := parseInt64Field(entry[0])
		if err != nil {
			return nil, fmt.Errorf("kline entry %d invalid open time: %w", i, err)
		}
		closePrice, err := parseFloatField(entry[4])
		if err != nil {
			return nil, fmt.Errorf("kline entry %d invalid close price: %w", i, err)
		}
		closeMS, err := parseInt64Field(entry[6])
		if err != nil {
			return nil, fmt.Errorf("kline entry %d invalid close time: %w", i, err)
		}

		klines = append(klines, Kline{
			OpenTime:  time.UnixMilli(openMS).UTC(),
			CloseTime: time.UnixMilli(closeMS).UTC(),
			Close:     closePrice,
		})
	}

	return klines, nil
}

// CreateMarketOrderQuote submits a market order using quote quantity.
func (c *Client) CreateMarketOrderQuote(ctx context.Context, symbol string, side Side, quoteOrderQty float64) (OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", strings.ToUpper(symbol))
	params.Set("side", string(side))
	params.Set("type", "MARKET")
	params.Set("quoteOrderQty", formatFloat(quoteOrderQty))

	body, err := c.request(ctx, http.MethodPost, "/api/v3/order", params, true)
	if err != nil {
		return OrderResponse{}, err
	}

	var response OrderResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return OrderResponse{}, fmt.Errorf("decode order response: %w", err)
	}
	return response, nil
}

func (c *Client) request(ctx context.Context, method, endpoint string, params url.Values, signed bool) ([]byte, error) {
	if params == nil {
		params = make(url.Values)
	}

	if signed {
		if c.apiKey == "" || c.apiSecret == "" {
			return nil, fmt.Errorf("missing API credentials for signed Binance request")
		}

		params.Set("recvWindow", strconv.Itoa(defaultRecvWindow))
		params.Set("timestamp", strconv.FormatInt(time.Now().UTC().UnixMilli(), 10))
		signature := c.sign(params.Encode())
		params.Set("signature", signature)
	}

	requestURL := c.baseURL + endpoint
	if encoded := params.Encode(); encoded != "" {
		requestURL += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if signed {
		req.Header.Set("X-MBX-APIKEY", c.apiKey)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s %s: %w", method, endpoint, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if res.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("%s %s failed with status %d: %s", method, endpoint, res.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}

func (c *Client) sign(payload string) string {
	hash := hmac.New(sha256.New, []byte(c.apiSecret))
	hash.Write([]byte(payload))
	return hex.EncodeToString(hash.Sum(nil))
}

func parseInt64Field(value interface{}) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case json.Number:
		return v.Int64()
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", value)
	}
}

func parseFloatField(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case json.Number:
		return v.Float64()
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("unsupported float type %T", value)
	}
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
