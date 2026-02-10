package app

import (
	"cloudpico-gateway/internal/ble"
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
	mqttClient, err := mqtt.NewClient(cfg)
	if err != nil {
		return err
	}

	go func() {
		// Connect to MQTT broker with retry and backoff
		if err := mqttClient.Connect(ctx); err != nil {
			slog.Error("mqtt connect failed", "error", err)
			return
		}
		defer mqttClient.Disconnect()
	}()
	bleListener := ble.NewListener(ble.Options{
		Adapter: "hci0",
		Filter: ble.Filter{
			LocalName:            "",
			CompanyID:            0xFFFF,
			ManufacturerDataPref: []byte{0x01, 0xD0},
		},
	})
	bleHandler := ble.NewBLESensorHandler(mqttClient)
	go func() {
		err := bleListener.Run(ctx, bleHandler.HandleMatch)
		if err != nil {
			slog.Warn("ble listener could not be initialized; gateway continues without BLE",
				"error", err,
			)
		}
	}()
	<-ctx.Done()

	slog.Info("gateway shutting down")
	return nil
}
