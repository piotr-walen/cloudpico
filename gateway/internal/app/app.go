package app

import (
	"cloudpico-gateway/internal/config"
	"cloudpico-gateway/internal/mqtt"
	"context"
	"log/slog"
)

func Run(ctx context.Context, cfg config.Config) error {
	slog.Info("initializing gateway",
		"mqtt_broker", cfg.MQTTBroker,
		"mqtt_port", cfg.MQTTPort,
		"mqtt_client_id", cfg.MQTTClientID,
	)

	// Initialize MQTT client
	mqttClient, err := mqtt.NewClient(cfg, slog.Default())
	if err != nil {
		return err
	}
	defer mqttClient.Disconnect()

	// Connect to MQTT broker with retry and backoff
	if err := mqttClient.Connect(ctx); err != nil {
		return err
	}

	slog.Info("gateway started, ready to publish telemetry")

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()

	slog.Info("gateway shutting down")
	return nil
}
