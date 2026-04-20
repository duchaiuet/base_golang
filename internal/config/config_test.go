package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("BINANCE_API_KEY", "")
	t.Setenv("BINANCE_API_SECRET", "")
	t.Setenv("BINANCE_SYMBOL", "ethusdt")
	t.Setenv("BINANCE_INTERVAL", "5m")
	t.Setenv("STRATEGY_MODE", "sma")
	t.Setenv("POLL_INTERVAL", "15s")
	t.Setenv("FAST_WINDOW", "5")
	t.Setenv("SLOW_WINDOW", "15")
	t.Setenv("MIN_SIGNAL_DELTA", "0.002")
	t.Setenv("QUOTE_ORDER_AMOUNT", "20")
	t.Setenv("REQUEST_TIMEOUT", "3s")
	t.Setenv("DRY_RUN", "true")
	t.Setenv("USE_TESTNET", "false")
	t.Setenv("KLINE_LIMIT", "22")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := cfg.Symbol, "ETHUSDT"; got != want {
		t.Fatalf("unexpected symbol: got %s want %s", got, want)
	}
	if got, want := cfg.FastWindow, 5; got != want {
		t.Fatalf("unexpected fast window: got %d want %d", got, want)
	}
	if got, want := cfg.SlowWindow, 15; got != want {
		t.Fatalf("unexpected slow window: got %d want %d", got, want)
	}
	if got, want := cfg.KlineLimit, 22; got != want {
		t.Fatalf("unexpected kline limit: got %d want %d", got, want)
	}
	if got, want := cfg.DryRun, true; got != want {
		t.Fatalf("unexpected dry-run setting: got %t want %t", got, want)
	}
	if got, want := cfg.UseTestnet, false; got != want {
		t.Fatalf("unexpected testnet setting: got %t want %t", got, want)
	}
	if got, want := cfg.BaseURL, defaultSpotBaseURL; got != want {
		t.Fatalf("unexpected base URL: got %s want %s", got, want)
	}
	if got, want := cfg.StrategyMode, StrategySMA; got != want {
		t.Fatalf("unexpected strategy mode: got %s want %s", got, want)
	}
}

func TestLoadRequiresCredentialsWhenNotDryRun(t *testing.T) {
	t.Setenv("DRY_RUN", "false")
	t.Setenv("BINANCE_API_KEY", "")
	t.Setenv("BINANCE_API_SECRET", "")
	t.Setenv("SLOW_WINDOW", "10")
	t.Setenv("FAST_WINDOW", "3")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected credential validation error, got nil")
	}
}

func TestLoadPredictiveModeDefaultsKlineLimit(t *testing.T) {
	t.Setenv("STRATEGY_MODE", "predictive")
	t.Setenv("PREDICT_TRAIN_WINDOW", "30")
	t.Setenv("KLINE_LIMIT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := cfg.StrategyMode, StrategyPredictive; got != want {
		t.Fatalf("unexpected strategy mode: got %s want %s", got, want)
	}
	if got, want := cfg.PredictTrainWindow, 30; got != want {
		t.Fatalf("unexpected train window: got %d want %d", got, want)
	}
	if got, want := cfg.KlineLimit, 36; got != want {
		t.Fatalf("unexpected kline limit: got %d want %d", got, want)
	}
}

func TestLoadRejectsInvalidStrategyMode(t *testing.T) {
	t.Setenv("STRATEGY_MODE", "random-forest")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected strategy mode validation error, got nil")
	}
}
