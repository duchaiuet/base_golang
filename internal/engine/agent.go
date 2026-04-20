package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/base_golang/binance-agent/internal/binance"
	"github.com/base_golang/binance-agent/internal/config"
	"github.com/base_golang/binance-agent/internal/strategy"
)

type positionState string

const (
	positionFlat positionState = "FLAT"
	positionLong positionState = "LONG"
)

// Client describes market-data and order APIs needed by the agent.
type Client interface {
	GetKlines(ctx context.Context, symbol, interval string, limit int) ([]binance.Kline, error)
	CreateMarketOrderQuote(ctx context.Context, symbol string, side binance.Side, quoteOrderQty float64) (binance.OrderResponse, error)
}

// Agent runs the trading strategy and executes orders.
type Agent struct {
	cfg      config.Config
	client   Client
	logger   *log.Logger
	position positionState
}

// New builds a new trading agent instance.
func New(cfg config.Config, client Client, logger *log.Logger) *Agent {
	return &Agent{
		cfg:      cfg,
		client:   client,
		logger:   logger,
		position: positionFlat,
	}
}

// Run starts the polling loop until context cancellation.
func (a *Agent) Run(ctx context.Context) error {
	a.logger.Printf("starting trading agent: %s", a.cfg.RedactedSummary())

	if err := a.Tick(ctx); err != nil {
		a.logger.Printf("initial tick failed: %v", err)
	}

	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logger.Printf("stopping agent: %v", ctx.Err())
			return nil
		case <-ticker.C:
			if err := a.Tick(ctx); err != nil {
				a.logger.Printf("tick failed: %v", err)
			}
		}
	}
}

// Tick executes one strategy iteration.
func (a *Agent) Tick(ctx context.Context) error {
	klines, err := a.client.GetKlines(ctx, a.cfg.Symbol, a.cfg.Interval, a.cfg.KlineLimit)
	if err != nil {
		return fmt.Errorf("get klines: %w", err)
	}
	if len(klines) == 0 {
		return fmt.Errorf("no kline data returned")
	}

	prices := make([]float64, 0, len(klines))
	for _, k := range klines {
		prices = append(prices, k.Close)
	}

	signal, err := strategy.SMA(prices, a.cfg.FastWindow, a.cfg.SlowWindow, a.cfg.MinSignalDelta)
	if err != nil {
		return fmt.Errorf("compute signal: %w", err)
	}

	a.logger.Printf(
		"signal=%s last_price=%.6f fast_sma=%.6f slow_sma=%.6f delta=%.6f position=%s",
		signal.Decision,
		signal.LastPrice,
		signal.FastSMA,
		signal.SlowSMA,
		signal.Delta,
		a.position,
	)

	switch signal.Decision {
	case strategy.DecisionBuy:
		if a.position == positionLong {
			a.logger.Printf("skip BUY: already in LONG position")
			return nil
		}
		return a.executeOrder(ctx, binance.SideBuy)
	case strategy.DecisionSell:
		if a.position == positionFlat {
			a.logger.Printf("skip SELL: no active LONG position")
			return nil
		}
		return a.executeOrder(ctx, binance.SideSell)
	default:
		return nil
	}
}

func (a *Agent) executeOrder(ctx context.Context, side binance.Side) error {
	if a.cfg.DryRun {
		a.logger.Printf(
			"[dry-run] would place %s MARKET order symbol=%s quoteOrderQty=%.6f",
			side,
			a.cfg.Symbol,
			a.cfg.QuoteOrderAmount,
		)
		a.updatePosition(side)
		return nil
	}

	resp, err := a.client.CreateMarketOrderQuote(ctx, a.cfg.Symbol, side, a.cfg.QuoteOrderAmount)
	if err != nil {
		return fmt.Errorf("place %s order: %w", side, err)
	}

	a.logger.Printf(
		"order placed side=%s symbol=%s order_id=%d status=%s executed_qty=%s quote_qty=%s transact_time=%d",
		side,
		resp.Symbol,
		resp.OrderID,
		resp.Status,
		resp.ExecutedQty,
		resp.CummulativeQuoteQty,
		resp.TransactTime,
	)
	a.updatePosition(side)
	return nil
}

func (a *Agent) updatePosition(side binance.Side) {
	switch side {
	case binance.SideBuy:
		a.position = positionLong
	case binance.SideSell:
		a.position = positionFlat
	}
}
