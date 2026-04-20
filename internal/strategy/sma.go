package strategy

import (
	"fmt"
	"math"
)

// Decision indicates whether to buy, sell, or hold.
type Decision string

const (
	DecisionHold Decision = "HOLD"
	DecisionBuy  Decision = "BUY"
	DecisionSell Decision = "SELL"
)

// Signal contains the strategy output and supporting values.
type Signal struct {
	Decision  Decision
	FastSMA   float64
	SlowSMA   float64
	Delta     float64
	LastPrice float64
}

// SMA computes buy/sell/hold using fast and slow simple moving averages.
func SMA(prices []float64, fastWindow, slowWindow int, minDelta float64) (Signal, error) {
	if len(prices) < slowWindow {
		return Signal{}, fmt.Errorf("insufficient prices: need at least %d, have %d", slowWindow, len(prices))
	}
	if fastWindow < 2 {
		return Signal{}, fmt.Errorf("fastWindow must be >= 2")
	}
	if slowWindow <= fastWindow {
		return Signal{}, fmt.Errorf("slowWindow must be > fastWindow")
	}
	if minDelta < 0 {
		return Signal{}, fmt.Errorf("minDelta must be non-negative")
	}

	fast := avg(prices[len(prices)-fastWindow:])
	slow := avg(prices[len(prices)-slowWindow:])
	lastPrice := prices[len(prices)-1]

	delta := 0.0
	if slow != 0 {
		delta = (fast - slow) / slow
	}

	decision := DecisionHold
	switch {
	case delta >= minDelta:
		decision = DecisionBuy
	case delta <= -minDelta:
		decision = DecisionSell
	}

	return Signal{
		Decision:  decision,
		FastSMA:   fast,
		SlowSMA:   slow,
		Delta:     delta,
		LastPrice: lastPrice,
	}, nil
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, v := range values {
		total += v
	}
	mean := total / float64(len(values))
	if math.IsNaN(mean) || math.IsInf(mean, 0) {
		return 0
	}
	return mean
}
