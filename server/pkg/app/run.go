package app

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"time"

	"cloudpico-server/pkg/config"
	"cloudpico-server/pkg/db"
	"cloudpico-server/pkg/httpapi"
)

func Run(ctx context.Context, cfg config.Config) error {
	dbConn, err := db.Open(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close(dbConn) }()

	var ok int
	if err := dbConn.QueryRow(`SELECT 1`).Scan(&ok); err != nil {
		log.Fatal(err)
	}

	slog.Info("DB OK", "ok", ok)

	srv := httpapi.NewServer(cfg)

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
