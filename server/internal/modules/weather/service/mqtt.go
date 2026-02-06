package service

import (
	"log/slog"

	"cloudpico-server/internal/modules/weather/repository"
	"cloudpico-server/internal/mqtt"
	cloudpico_shared "cloudpico-shared/types"
)

// registerMQTTHandler sets up the weather module's MQTT message handler
func registerMQTTHandler(subscriber mqtt.MQTTSubscriber, repo repository.WeatherRepository, logger *slog.Logger) {
	subscriber.SetMessageHandler(func(telemetry cloudpico_shared.Telemetry) error {
		logger.Debug("processing telemetry message",
			"station_id", telemetry.StationID,
			"timestamp", telemetry.Timestamp,
		)

		err := repo.InsertReading(
			telemetry.StationID,
			telemetry.Timestamp,
			telemetry.Temperature,
			telemetry.Humidity,
			telemetry.Pressure,
		)

		if err != nil {
			logger.Error("failed to insert reading",
				"station_id", telemetry.StationID,
				"error", err,
			)
			return err
		}

		logger.Debug("successfully stored telemetry",
			"station_id", telemetry.StationID,
		)
		return nil
	})
}
