// server/cmd/main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
)

const (
	appName = "server"
	// Default version is "dev" if not set with -ldflags "-X main.version=..."
	version = "dev"
)

type Config struct {
	AppEnv   string
	LogLevel slog.Level
}

func main() {
	cfg, err := loadConfigFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)

	slog.Info("starting",
		"app", appName,
		"version", version,
		"env", cfg.AppEnv,
		"log_level", cfg.LogLevel.String(),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("run failed", "err", err)
		os.Exit(1)
	}

	slog.Info("shutting down")
}

func loadConfigFromEnv() (Config, error) {
	appEnv := strings.TrimSpace(os.Getenv("APP_ENV"))
	if appEnv == "" {
		appEnv = "dev"
	}

	switch appEnv {
	case "dev", "prod":
	default:
		return Config{}, fmt.Errorf("invalid APP_ENV %q (allowed: dev, prod)", appEnv)
	}

	logLevelStr := strings.TrimSpace(os.Getenv("LOG_LEVEL"))
	if logLevelStr == "" {
		logLevelStr = "info"
	}

	level, err := parseLogLevel(logLevelStr)
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppEnv:   appEnv,
		LogLevel: level,
	}, nil
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid LOG_LEVEL %q (allowed: debug, info, warn, error)", s)
	}
}

func newLogger(cfg Config) *slog.Logger {
	if version == "dev" {
		h := tint.NewHandler(os.Stdout, &tint.Options{
			Level:      cfg.LogLevel,
			AddSource:  true,
			TimeFormat: time.Kitchen,
		})
		return slog.New(h).With(
			"app", appName,
		)
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})
	return slog.New(h).With(
		"app", appName,
		"version", version,
		"env", cfg.AppEnv,
	)

}

func run(ctx context.Context) error {
	slog.Info("started")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			slog.Info("tick")
		}
	}
}
