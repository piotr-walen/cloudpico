package main

import (
	"cloudpico-gateway/internal/app"
	"cloudpico-gateway/internal/config"
	"cloudpico-gateway/internal/logging"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

var version = "dev"
var appName = "cloudpico-gateway"

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	logger := logging.New(cfg, version, appName)
	slog.SetDefault(logger)

	slog.Info("starting",
		"app", appName,
		"version", version,
		"env", cfg.AppEnv,
		"log_level", cfg.LogLevel.String(),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("run failed", "err", err)
		os.Exit(1)
	}

	slog.Info("shutting down")
}
