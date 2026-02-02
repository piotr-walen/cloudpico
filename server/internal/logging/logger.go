package logging

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"

	"cloudpico-server/internal/config"
)

func New(cfg config.Config, version string, appName string) *slog.Logger {
	if version == "dev" {
		h := tint.NewHandler(os.Stdout, &tint.Options{
			Level:      cfg.LogLevel,
			AddSource:  true,
			TimeFormat: time.Kitchen,
		})
		return slog.New(h).With("app", appName)
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
