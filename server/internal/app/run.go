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
	weatherviews "cloudpico-server/internal/modules/weather/views"
	"cloudpico-server/internal/mqtt"
)

func Run(ctx context.Context, cfg config.Config) error {
	slog.Info("config loaded",
		"appEnv", cfg.AppEnv,
		"logLevel", cfg.LogLevel.String(),
		"httpAddr", cfg.HTTPAddr,
		"staticDir", cfg.StaticDir,
		"sqliteDriver", cfg.SQLiteDriver,
		"sqlitePath", cfg.SQLitePath,
		"sqliteMaxOpenConns", cfg.SQLiteMaxOpenConns,
		"sqliteMaxIdleConns", cfg.SQLiteMaxIdleConns,
		"sqliteConnMaxLifetime", cfg.SQLiteConnMaxLifetime,
		"mqttBroker", cfg.MQTTBroker,
		"mqttPort", cfg.MQTTPort,
		"mqttTopic", cfg.MQTTTopic,
	)
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

	// Set MQTT handler before Connect so OnConnectHandler can subscribe immediately.
	// The broker may send queued messages right after CONNACK; we must be subscribed
	// before that to receive them.
	mux := httpapi.NewMux(dbConn, cfg.StaticDir)
	if err := weatherviews.LoadTemplates(); err != nil {
		return err
	}
	mqttSubscriber := mqtt.NewSubscriber(cfg)
	weather.RegisterFeature(mux, dbConn, mqttSubscriber)

	if err := mqttSubscriber.Connect(ctx); err != nil {
		slog.Warn("mqtt connection failed (continuing without mqtt)", "error", err)
		return err
	}

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

	slog.Info("mqtt disconnecting")
	mqttSubscriber.Disconnect()

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
