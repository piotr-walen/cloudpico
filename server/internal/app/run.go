package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"cloudpico-server/internal/config"
	db "cloudpico-server/internal/db"
	httpapi "cloudpico-server/internal/httpapi"
	weather "cloudpico-server/internal/modules/weather"
)

func Run(ctx context.Context, cfg config.Config) error {
	dbConn, err := db.Open(cfg)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := db.Close(dbConn)
		if closeErr != nil {
			slog.Error("db close", "error", closeErr)
		}
	}()

	var ok int
	err = dbConn.QueryRow(`SELECT 1`).Scan(&ok)
	if err != nil {
		return err
	}
	if ok != 1 {
		return errors.New("database connection failed")
	}
	slog.Info("database connection successful")

	mux := httpapi.NewMux(dbConn, "static")

	weather.RegisterFeature(mux, dbConn)

	srv := httpapi.NewServer(cfg, mux)

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

	err = <-errCh
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return ctx.Err()
}
