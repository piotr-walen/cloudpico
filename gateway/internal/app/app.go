package app

import (
	"cloudpico-gateway/internal/ble"
	"cloudpico-gateway/internal/config"
	"cloudpico-gateway/internal/mqtt"
	"cloudpico-gateway/internal/utils"
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

	slog.Info("gateway started (no mqtt publishing yet), starting BLE listener")

	// Start BLE listener (no MQTT publish yet)
	bleListener := ble.NewListener(ble.Options{
		Adapter: "hci0",
		Filter: ble.Filter{
			LocalName:            "pico2w-done", // set "" to ignore name
			CompanyID:            0xFFFF,
			ManufacturerDataPref: []byte{0x01, 0xD0},
		},
		Debug: false,
	}, slog.Default())

	go func() {
		err := bleListener.Run(ctx, func(m ble.Match) {
			// For now: just log. No MQTT.
			slog.Info("ble: DONE beacon received",
				"addr", m.Address,
				"rssi", m.RSSI,
				"name", m.LocalName,
				"company", slog.StringValue("0x"+utils.Hex4(m.CompanyID)),
				"data", utils.BytesToHex(m.Data),
			)
		})
		if err != nil {
			slog.Warn("ble listener could not be initialized; gateway continues without BLE",
				"error", err,
			)
		}
	}()

	// Wait for context cancellation (shutdown signal)
	<-ctx.Done()

	slog.Info("gateway shutting down")
	return nil
}
