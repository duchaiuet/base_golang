package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSpotBaseURL    = "https://api.binance.com"
	defaultTestnetBaseURL = "https://testnet.binance.vision"
)

// StrategyMode selects which decision engine to use.
type StrategyMode string

const (
	StrategyPredictive StrategyMode = "predictive"
	StrategySMA        StrategyMode = "sma"
)

// Config stores runtime parameters for the trading agent.
type Config struct {
	APIKey             string
	APISecret          string
	BaseURL            string
	Symbol             string
	Interval           string
	StrategyMode       StrategyMode
	PollInterval       time.Duration
	FastWindow         int
	SlowWindow         int
	PredictTrainWindow int
	MinSignalDelta     float64
	QuoteOrderAmount   float64
	KlineLimit         int
	RequestTimeout     time.Duration
	DryRun             bool
	UseTestnet         bool
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		APIKey:             strings.TrimSpace(os.Getenv("BINANCE_API_KEY")),
		APISecret:          strings.TrimSpace(os.Getenv("BINANCE_API_SECRET")),
		Symbol:             strings.ToUpper(envOrDefault("BINANCE_SYMBOL", "BTCUSDT")),
		Interval:           envOrDefault("BINANCE_INTERVAL", "1m"),
		StrategyMode:       StrategyPredictive,
		PollInterval:       30 * time.Second,
		FastWindow:         7,
		SlowWindow:         25,
		PredictTrainWindow: 120,
		MinSignalDelta:     0.001,
		QuoteOrderAmount:   25,
		RequestTimeout:     10 * time.Second,
		DryRun:             true,
		UseTestnet:         true,
	}

	var err error

	cfg.PollInterval, err = durationFromEnv("POLL_INTERVAL", cfg.PollInterval)
	if err != nil {
		return Config{}, err
	}
	cfg.StrategyMode = StrategyMode(strings.ToLower(envOrDefault("STRATEGY_MODE", string(cfg.StrategyMode))))
	cfg.FastWindow, err = intFromEnv("FAST_WINDOW", cfg.FastWindow)
	if err != nil {
		return Config{}, err
	}
	cfg.SlowWindow, err = intFromEnv("SLOW_WINDOW", cfg.SlowWindow)
	if err != nil {
		return Config{}, err
	}
	cfg.PredictTrainWindow, err = intFromEnv("PREDICT_TRAIN_WINDOW", cfg.PredictTrainWindow)
	if err != nil {
		return Config{}, err
	}
	cfg.MinSignalDelta, err = floatFromEnv("MIN_SIGNAL_DELTA", cfg.MinSignalDelta)
	if err != nil {
		return Config{}, err
	}
	cfg.QuoteOrderAmount, err = floatFromEnv("QUOTE_ORDER_AMOUNT", cfg.QuoteOrderAmount)
	if err != nil {
		return Config{}, err
	}
	cfg.RequestTimeout, err = durationFromEnv("REQUEST_TIMEOUT", cfg.RequestTimeout)
	if err != nil {
		return Config{}, err
	}
	cfg.DryRun, err = boolFromEnv("DRY_RUN", cfg.DryRun)
	if err != nil {
		return Config{}, err
	}
	cfg.UseTestnet, err = boolFromEnv("USE_TESTNET", cfg.UseTestnet)
	if err != nil {
		return Config{}, err
	}

	defaultKlineLimit := cfg.SlowWindow + 5
	if cfg.StrategyMode == StrategyPredictive {
		defaultKlineLimit = cfg.PredictTrainWindow + 6
	}
	cfg.KlineLimit, err = intFromEnv("KLINE_LIMIT", defaultKlineLimit)
	if err != nil {
		return Config{}, err
	}

	if customURL := strings.TrimSpace(os.Getenv("BINANCE_BASE_URL")); customURL != "" {
		cfg.BaseURL = customURL
	} else if cfg.UseTestnet {
		cfg.BaseURL = defaultTestnetBaseURL
	} else {
		cfg.BaseURL = defaultSpotBaseURL
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// RedactedSummary returns a safe-to-log one-line config summary.
func (c Config) RedactedSummary() string {
	keySet := c.APIKey != "" && c.APISecret != ""
	return fmt.Sprintf(
		"symbol=%s interval=%s strategy=%s poll=%s fast=%d slow=%d train=%d delta=%.4f quote_amount=%.4f dry_run=%t testnet=%t auth=%t base_url=%s",
		c.Symbol,
		c.Interval,
		c.StrategyMode,
		c.PollInterval,
		c.FastWindow,
		c.SlowWindow,
		c.PredictTrainWindow,
		c.MinSignalDelta,
		c.QuoteOrderAmount,
		c.DryRun,
		c.UseTestnet,
		keySet,
		c.BaseURL,
	)
}

func (c Config) validate() error {
	if c.Symbol == "" {
		return fmt.Errorf("BINANCE_SYMBOL cannot be empty")
	}
	if c.Interval == "" {
		return fmt.Errorf("BINANCE_INTERVAL cannot be empty")
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("POLL_INTERVAL must be greater than zero")
	}
	switch c.StrategyMode {
	case StrategySMA:
		if c.FastWindow < 2 {
			return fmt.Errorf("FAST_WINDOW must be at least 2 for STRATEGY_MODE=sma")
		}
		if c.SlowWindow <= c.FastWindow {
			return fmt.Errorf("SLOW_WINDOW must be greater than FAST_WINDOW for STRATEGY_MODE=sma")
		}
	case StrategyPredictive:
		if c.PredictTrainWindow < 5 {
			return fmt.Errorf("PREDICT_TRAIN_WINDOW must be at least 5 for STRATEGY_MODE=predictive")
		}
	default:
		return fmt.Errorf("STRATEGY_MODE must be one of: %s, %s", StrategyPredictive, StrategySMA)
	}
	if c.MinSignalDelta < 0 {
		return fmt.Errorf("MIN_SIGNAL_DELTA must be non-negative")
	}
	if c.QuoteOrderAmount <= 0 {
		return fmt.Errorf("QUOTE_ORDER_AMOUNT must be greater than zero")
	}
	switch c.StrategyMode {
	case StrategySMA:
		if c.KlineLimit < c.SlowWindow {
			return fmt.Errorf("KLINE_LIMIT must be >= SLOW_WINDOW for STRATEGY_MODE=sma")
		}
	case StrategyPredictive:
		if c.KlineLimit < c.PredictTrainWindow+4 {
			return fmt.Errorf("KLINE_LIMIT must be >= PREDICT_TRAIN_WINDOW + 4 for STRATEGY_MODE=predictive")
		}
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("REQUEST_TIMEOUT must be greater than zero")
	}
	if !c.DryRun && (c.APIKey == "" || c.APISecret == "") {
		return fmt.Errorf("BINANCE_API_KEY and BINANCE_API_SECRET are required when DRY_RUN=false")
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func boolFromEnv(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("invalid %s: %w", key, err)
	}
	return value, nil
}

func intFromEnv(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return value, nil
}

func floatFromEnv(key string, fallback float64) (float64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return value, nil
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return value, nil
}
