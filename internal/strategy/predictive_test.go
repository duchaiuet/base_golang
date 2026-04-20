package strategy

import (
	"math"
	"testing"
)

func TestPredictiveBuySignalOnUptrend(t *testing.T) {
	t.Parallel()

	prices := makeTrendPrices(100, 0.002, 80)
	signal, err := Predictive(prices, 30, 0.0008)
	if err != nil {
		t.Fatalf("Predictive() unexpected error: %v", err)
	}

	if got, want := signal.Decision, DecisionBuy; got != want {
		t.Fatalf("unexpected decision: got %s want %s", got, want)
	}
	if signal.ModelConfidence <= 0 {
		t.Fatalf("expected positive confidence, got %.6f", signal.ModelConfidence)
	}
}

func TestPredictiveSellSignalOnDowntrend(t *testing.T) {
	t.Parallel()

	prices := makeTrendPrices(100, -0.002, 80)
	signal, err := Predictive(prices, 30, 0.0008)
	if err != nil {
		t.Fatalf("Predictive() unexpected error: %v", err)
	}

	if got, want := signal.Decision, DecisionSell; got != want {
		t.Fatalf("unexpected decision: got %s want %s", got, want)
	}
}

func TestPredictiveRejectsInsufficientData(t *testing.T) {
	t.Parallel()

	_, err := Predictive([]float64{100, 101, 102, 103, 104, 105}, 10, 0.001)
	if err == nil {
		t.Fatalf("expected insufficient data error, got nil")
	}
}

func TestPredictiveFallbackWhenSingular(t *testing.T) {
	t.Parallel()

	// Constant growth creates near-identical feature rows and can trigger
	// singular normal equations in OLS. Strategy should fallback safely.
	prices := makeTrendPrices(100, 0.001, 50)
	signal, err := Predictive(prices, 20, 0.0001)
	if err != nil {
		t.Fatalf("Predictive() unexpected error with fallback: %v", err)
	}

	if signal.ModelConfidence < 0 || signal.ModelConfidence > 1 {
		t.Fatalf("confidence out of range: %.6f", signal.ModelConfidence)
	}
	if math.IsNaN(signal.PredictedReturn) || math.IsInf(signal.PredictedReturn, 0) {
		t.Fatalf("invalid predicted return: %.6f", signal.PredictedReturn)
	}
}

func makeTrendPrices(start, step float64, n int) []float64 {
	out := make([]float64, 0, n)
	value := start
	for i := 0; i < n; i++ {
		if i == 0 {
			out = append(out, value)
			continue
		}
		value = value * (1 + step)
		out = append(out, value)
	}
	return out
}
