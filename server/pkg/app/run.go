package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"cloudpico-server/pkg/config"
	"cloudpico-server/pkg/httpapi"
)

func Run(ctx context.Context, cfg config.Config) error {
	srv := httpapi.NewServer(cfg.HTTPAddr)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http listening", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.Info("http shutting down")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	err := <-errCh
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return ctx.Err()
}
