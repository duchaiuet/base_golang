package engine

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/base_golang/binance-agent/internal/binance"
	"github.com/base_golang/binance-agent/internal/config"
)

type orderCall struct {
	symbol        string
	side          binance.Side
	quoteOrderQty float64
}

type mockClient struct {
	klines         [][]binance.Kline
	orderResponses []binance.OrderResponse
	getErr         error
	orderErr       error

	getCalls   int
	orderCalls []orderCall
}

func (m *mockClient) GetKlines(_ context.Context, _, _ string, _ int) ([]binance.Kline, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if len(m.klines) == 0 {
		return nil, nil
	}
	if m.getCalls >= len(m.klines) {
		last := m.klines[len(m.klines)-1]
		m.getCalls++
		return last, nil
	}
	klines := m.klines[m.getCalls]
	m.getCalls++
	return klines, nil
}

func (m *mockClient) CreateMarketOrderQuote(_ context.Context, symbol string, side binance.Side, quoteOrderQty float64) (binance.OrderResponse, error) {
	if m.orderErr != nil {
		return binance.OrderResponse{}, m.orderErr
	}
	m.orderCalls = append(m.orderCalls, orderCall{
		symbol:        symbol,
		side:          side,
		quoteOrderQty: quoteOrderQty,
	})

	if len(m.orderResponses) == 0 {
		return binance.OrderResponse{Symbol: symbol, Status: "FILLED", OrderID: int64(len(m.orderCalls))}, nil
	}
	idx := len(m.orderCalls) - 1
	if idx >= len(m.orderResponses) {
		return m.orderResponses[len(m.orderResponses)-1], nil
	}
	return m.orderResponses[idx], nil
}

func TestTickPlacesBuyOrder(t *testing.T) {
	t.Parallel()

	client := &mockClient{
		klines: [][]binance.Kline{
			makeKlines([]float64{10, 10.2, 10.5, 11.0, 11.4}),
		},
	}
	agent := New(testConfig(false), client, log.New(io.Discard, "", 0))

	if err := agent.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() unexpected error: %v", err)
	}

	if got, want := len(client.orderCalls), 1; got != want {
		t.Fatalf("unexpected order count: got %d want %d", got, want)
	}
	call := client.orderCalls[0]
	if got, want := call.side, binance.SideBuy; got != want {
		t.Fatalf("unexpected side: got %s want %s", got, want)
	}
	if got, want := call.symbol, "BTCUSDT"; got != want {
		t.Fatalf("unexpected symbol: got %s want %s", got, want)
	}
	if got, want := call.quoteOrderQty, 25.0; got != want {
		t.Fatalf("unexpected quote qty: got %.4f want %.4f", got, want)
	}
}

func TestTickSkipsSellWhenFlat(t *testing.T) {
	t.Parallel()

	client := &mockClient{
		klines: [][]binance.Kline{
			makeKlines([]float64{11.5, 11.2, 11.0, 10.7, 10.5}),
		},
	}
	agent := New(testConfig(false), client, log.New(io.Discard, "", 0))

	if err := agent.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() unexpected error: %v", err)
	}

	if got := len(client.orderCalls); got != 0 {
		t.Fatalf("expected no orders while flat on sell signal, got %d", got)
	}
}

func TestTickPlacesSellOrderAfterBuy(t *testing.T) {
	t.Parallel()

	client := &mockClient{
		klines: [][]binance.Kline{
			makeKlines([]float64{10, 10.2, 10.5, 11.0, 11.4}),
			makeKlines([]float64{11.5, 11.2, 11.0, 10.7, 10.5}),
		},
	}
	agent := New(testConfig(false), client, log.New(io.Discard, "", 0))

	if err := agent.Tick(context.Background()); err != nil {
		t.Fatalf("first Tick() unexpected error: %v", err)
	}
	if err := agent.Tick(context.Background()); err != nil {
		t.Fatalf("second Tick() unexpected error: %v", err)
	}

	if got, want := len(client.orderCalls), 2; got != want {
		t.Fatalf("unexpected order count: got %d want %d", got, want)
	}
	if got, want := client.orderCalls[0].side, binance.SideBuy; got != want {
		t.Fatalf("unexpected first side: got %s want %s", got, want)
	}
	if got, want := client.orderCalls[1].side, binance.SideSell; got != want {
		t.Fatalf("unexpected second side: got %s want %s", got, want)
	}
}

func TestTickDryRunUpdatesPositionWithoutPlacingOrder(t *testing.T) {
	t.Parallel()

	client := &mockClient{
		klines: [][]binance.Kline{
			makeKlines([]float64{10, 10.2, 10.5, 11.0, 11.4}),
		},
	}
	agent := New(testConfig(true), client, log.New(io.Discard, "", 0))

	if err := agent.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() unexpected error: %v", err)
	}

	if got := len(client.orderCalls); got != 0 {
		t.Fatalf("expected no orders in dry-run mode, got %d", got)
	}
	if got, want := agent.position, positionLong; got != want {
		t.Fatalf("unexpected position: got %s want %s", got, want)
	}
}

func testConfig(dryRun bool) config.Config {
	return config.Config{
		Symbol:             "BTCUSDT",
		Interval:           "1m",
		StrategyMode:       config.StrategySMA,
		PollInterval:       5 * time.Second,
		FastWindow:         2,
		SlowWindow:         4,
		PredictTrainWindow: 10,
		MinSignalDelta:     0.005,
		QuoteOrderAmount:   25,
		KlineLimit:         6,
		RequestTimeout:     5 * time.Second,
		DryRun:             dryRun,
		UseTestnet:         true,
		BaseURL:            "https://testnet.binance.vision",
	}
}

func makeKlines(prices []float64) []binance.Kline {
	out := make([]binance.Kline, 0, len(prices))
	now := time.Now().UTC()
	for i, price := range prices {
		openTime := now.Add(time.Duration(i) * time.Minute)
		out = append(out, binance.Kline{
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Close:     price,
		})
	}
	return out
}
