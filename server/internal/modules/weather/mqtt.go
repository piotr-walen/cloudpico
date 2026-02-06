package weather

import (
	"log/slog"

	"cloudpico-server/internal/modules/weather/repository"
	cloudpico_shared "cloudpico-shared/types"
)

// MQTTSubscriber interface for attaching message handlers
type MQTTSubscriber interface {
	SetMessageHandler(handler func(telemetry cloudpico_shared.Telemetry) error)
}

// registerMQTTHandler sets up the weather module's MQTT message handler
func registerMQTTHandler(subscriber MQTTSubscriber, repo repository.WeatherRepository, logger *slog.Logger) {
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
