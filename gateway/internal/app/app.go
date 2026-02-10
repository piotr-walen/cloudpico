package app

import (
	"cloudpico-gateway/internal/ble"
	"cloudpico-gateway/internal/config"
	"cloudpico-gateway/internal/mqtt"
	"cloudpico-gateway/internal/sensor"
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

	bleListener := ble.NewListener(ble.Options{
		Adapter: "hci0",
		Filter: ble.Filter{
			LocalName:            "",
			CompanyID:            0xFFFF,
			ManufacturerDataPref: []byte{0x01, 0xD0},
		},
	}, slog.Default())

	bleHandler := ble.NewBLESensorHandler(mqttClient)
	bleHandler.StartListener(ctx, bleListener)

	go func() {
		err := sensor.Run(ctx, cfg, mqttClient)
		if err != nil {
			slog.Warn("internal sensor could not be initialized; gateway continues without sensor",
				"error", err,
			)
		}
	}()
	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()

	slog.Info("gateway shutting down")
	return nil
}
