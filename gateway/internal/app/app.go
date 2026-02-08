package app

import (
	"cloudpico-gateway/internal/config"
	"cloudpico-gateway/internal/mqtt"
	cloudpico_shared "cloudpico-shared/types"
	"context"
	"fmt"
	"log/slog"
	"time"
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

	// Start publishing telemetry data periodically
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		sequence := 0
		stationID := "1" // Default station ID, could be made configurable

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sequence++
				temp := 22.5 + float64(sequence%10)*0.1       // Sample temperature
				humidity := 45.0 + float64(sequence%20)*0.5   // Sample humidity
				pressure := 1013.25 + float64(sequence%5)*0.1 // Sample pressure
				battery := 3.7 + float64(sequence%10)*0.01    // Sample battery voltage

				telemetry := cloudpico_shared.Telemetry{
					StationID:   stationID,
					Timestamp:   time.Now(),
					Temperature: &temp,
					Humidity:    &humidity,
					Pressure:    &pressure,
					Battery:     &battery,
					Sequence:    &sequence,
				}

				slog.Info("publishing telemetry",
					"station_id", stationID,
					"timestamp", fmt.Sprintf("%s", telemetry.Timestamp),
					"temperature", fmt.Sprintf("%f °C", *telemetry.Temperature),
					"humidity", fmt.Sprintf("%f %%", *telemetry.Humidity),
					"pressure", fmt.Sprintf("%f hPa", *telemetry.Pressure),
					"battery", fmt.Sprintf("%f V", *telemetry.Battery),
					"sequence", fmt.Sprintf("%d", *telemetry.Sequence),
				)
				if err := mqttClient.PublishTelemetry(telemetry); err != nil {
					slog.Error("failed to publish telemetry", "error", err)
				} else {
					slog.Debug("published telemetry",
						"station_id", stationID,
						"sequence", sequence,
					)
				}
			}
		}
	}()

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()

	slog.Info("gateway shutting down")
	return nil
}
