# Binance Trading Agent (Go)

Binance Spot trading bot in Go with a default **prediction-based strategy**.

The agent:

- Polls recent klines for a symbol.
- Runs one of two strategy modes:
  - `predictive` (default): online linear model predicts next-candle return.
  - `sma`: classic fast/slow SMA crossover.
- Places market orders (or logs intended orders in dry-run mode).
- Tracks in-memory position state (`FLAT` / `LONG`) to avoid repeated same-side orders.

## Strategy modes

### 1) Predictive (default)

The predictive strategy refits a lightweight OLS model on each tick using recent candles.

Target:

- next candle return

Features:

- lag-1 return
- lag-2 return
- 3-candle momentum

Decision rule:

- `BUY` when predicted return `>= MIN_SIGNAL_DELTA`
- `SELL` when predicted return `<= -MIN_SIGNAL_DELTA`
- otherwise `HOLD`

If the regression matrix becomes singular in flat/degenerate windows, the bot falls back to a conservative proxy (latest observed return) with low confidence.

### 2) SMA

Fast/slow moving average crossover:

- `BUY` when `(fast - slow)/slow >= MIN_SIGNAL_DELTA`
- `SELL` when `(fast - slow)/slow <= -MIN_SIGNAL_DELTA`
- otherwise `HOLD`

## Safety defaults

The bot is intentionally safe by default:

- `DRY_RUN=true` (no real orders sent)
- `USE_TESTNET=true`
- API credentials required only when `DRY_RUN=false`

Use testnet first and validate logs before enabling live orders.

## Project layout

- `cmd/trader/main.go` - application entrypoint.
- `internal/config` - environment-driven runtime configuration.
- `internal/binance` - minimal Binance REST client.
- `internal/strategy` - predictive and SMA strategy logic.
- `internal/engine` - polling + signal + order execution loop.

## Configuration

Set environment variables before running.

| Variable | Default | Description |
| --- | --- | --- |
| `BINANCE_API_KEY` | `""` | Binance API key (required when `DRY_RUN=false`). |
| `BINANCE_API_SECRET` | `""` | Binance API secret (required when `DRY_RUN=false`). |
| `BINANCE_BASE_URL` | derived | Optional custom API base URL; overrides testnet/mainnet defaults. |
| `USE_TESTNET` | `true` | Uses `https://testnet.binance.vision` when true, else `https://api.binance.com`. |
| `DRY_RUN` | `true` | If true, logs intended orders without sending them. |
| `BINANCE_SYMBOL` | `BTCUSDT` | Trading pair symbol. |
| `BINANCE_INTERVAL` | `1m` | Kline interval (e.g. `1m`, `5m`, `15m`). |
| `STRATEGY_MODE` | `predictive` | `predictive` or `sma`. |
| `PREDICT_TRAIN_WINDOW` | `120` | Number of recent predictive samples used for OLS fitting. |
| `FAST_WINDOW` | `7` | Fast SMA window size (`sma` mode only). |
| `SLOW_WINDOW` | `25` | Slow SMA window size (`sma` mode only, must be > fast). |
| `MIN_SIGNAL_DELTA` | `0.001` | Signal threshold (predicted return or SMA delta). |
| `QUOTE_ORDER_AMOUNT` | `25` | Quote asset amount used per market order. |
| `KLINE_LIMIT` | strategy-dependent | Candles fetched each tick. Defaults to `PREDICT_TRAIN_WINDOW+6` (predictive) or `SLOW_WINDOW+5` (sma). |
| `POLL_INTERVAL` | `30s` | Time between strategy ticks. |
| `REQUEST_TIMEOUT` | `10s` | HTTP timeout for Binance requests. |

## Run

```bash
go run ./cmd/trader
```

Example (predictive strategy, testnet dry run):

```bash
export STRATEGY_MODE=predictive
export USE_TESTNET=true
export DRY_RUN=true
export BINANCE_SYMBOL=BTCUSDT
go run ./cmd/trader
```

Example (SMA strategy):

```bash
export STRATEGY_MODE=sma
export FAST_WINDOW=7
export SLOW_WINDOW=25
export MIN_SIGNAL_DELTA=0.0015
go run ./cmd/trader
```

Example (testnet live orders):

```bash
export USE_TESTNET=true
export DRY_RUN=false
export BINANCE_API_KEY=your_testnet_api_key
export BINANCE_API_SECRET=your_testnet_api_secret
go run ./cmd/trader
```

## Tests

```bash
go test ./...
```

## Notes

- This is not financial advice.
- The predictive mode is an online lightweight model, not a guaranteed profitable AI.
- Add risk controls (max daily loss, max open exposure, stop logic, circuit breaker) before using real funds.