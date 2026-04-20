# Binance Trading Agent (Go)

Simple Binance Spot trading bot built in Go.

The agent:

- Polls recent klines for a symbol.
- Computes a fast/slow SMA crossover signal.
- Places market orders (or logs them in dry-run mode).
- Tracks a minimal in-memory position state (`FLAT`/`LONG`) to avoid repeated same-side orders.

## Safety defaults

The bot is intentionally configured to be safe by default:

- `DRY_RUN=true` by default (no real orders are placed).
- `USE_TESTNET=true` by default.
- API credentials are only required when `DRY_RUN=false`.

Use live mode only after validating behavior on testnet.

## Project layout

- `cmd/trader/main.go` - application entrypoint.
- `internal/config` - environment-driven runtime configuration.
- `internal/binance` - minimal Binance REST client.
- `internal/strategy` - SMA signal logic.
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
| `POLL_INTERVAL` | `30s` | Time between strategy ticks. |
| `FAST_WINDOW` | `7` | Fast SMA window size. |
| `SLOW_WINDOW` | `25` | Slow SMA window size (must be greater than fast window). |
| `MIN_SIGNAL_DELTA` | `0.001` | Relative threshold for buy/sell decision. |
| `QUOTE_ORDER_AMOUNT` | `25` | Quote asset amount used per market order. |
| `KLINE_LIMIT` | `SLOW_WINDOW+5` | Number of candles requested each tick. |
| `REQUEST_TIMEOUT` | `10s` | HTTP timeout for Binance requests. |

## Run

```bash
go run ./cmd/trader
```

Example (testnet dry run):

```bash
export USE_TESTNET=true
export DRY_RUN=true
export BINANCE_SYMBOL=BTCUSDT
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
- The strategy is intentionally simple and meant as a foundation for extension.
- Add risk controls (max daily loss, max open exposure, stop logic, symbol filters) before using with real funds.