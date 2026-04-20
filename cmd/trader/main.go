package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/base_golang/binance-agent/internal/binance"
	"github.com/base_golang/binance-agent/internal/config"
	"github.com/base_golang/binance-agent/internal/engine"
)

func main() {
	logger := log.New(os.Stdout, "[binance-agent] ", log.LstdFlags|log.Lmicroseconds|log.LUTC)

	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("failed to load config: %v", err)
	}

	client := binance.NewClient(cfg.BaseURL, cfg.APIKey, cfg.APISecret, cfg.RequestTimeout)
	agent := engine.New(cfg, client, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := agent.Run(ctx); err != nil {
		logger.Fatalf("agent exited with error: %v", err)
	}
}
