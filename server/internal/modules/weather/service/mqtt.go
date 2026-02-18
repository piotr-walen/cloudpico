package service

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"cloudpico-server/internal/modules/weather/repository"
	internalmqtt "cloudpico-server/internal/mqtt"
	cloudpico_shared "cloudpico-shared/types"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func validateTelemetry(t cloudpico_shared.Telemetry) error {
	// Validate required fields
	if t.StationID == "" {
		return fmt.Errorf("station_id is required")
	}

	if t.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	// Validate optional fields if present
	if t.Humidity != nil {
		if *t.Humidity < 0 || *t.Humidity > 100 {
			return fmt.Errorf("humidity_pct out of range: %f (must be 0-100)", *t.Humidity)
		}
	}

	if t.Pressure != nil {
		if *t.Pressure <= 0 {
			return fmt.Errorf("pressure_hpa must be positive: %f", *t.Pressure)
		}
	}

	// At least one sensor reading should be present
	if t.Temperature == nil && t.Humidity == nil && t.Pressure == nil {
		return fmt.Errorf("at least one sensor reading (temperature, humidity, or pressure) is required")
	}

	return nil
}

func parseTelemetry(payload []byte) (cloudpico_shared.Telemetry, error) {
	var telemetry cloudpico_shared.Telemetry
	if err := json.Unmarshal(payload, &telemetry); err != nil {
		return cloudpico_shared.Telemetry{}, err
	}
	return telemetry, nil
}

// formatOptFloat formats an optional float for logging; returns "-" if nil.
func formatOptFloat(p *float64, unit string) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%f %s", *p, unit)
}

// formatOptUint32 formats an optional uint32 for logging; returns "-" if nil.
func formatOptUint32(p *uint32) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *p)
}

// registerMQTTHandler sets up the weather module's MQTT message handler
func registerMQTTHandler(subscriber *internalmqtt.Subscriber, repo repository.WeatherRepository) {
	subscriber.SetMessageHandler(func(msg mqtt.Message) error {
		telemetry, err := parseTelemetry(msg.Payload())
		if err != nil {
			return err
		}

		if err := validateTelemetry(telemetry); err != nil {
			return err
		}

		slog.Info("inserting reading",
			"station_id", telemetry.StationID,
			"timestamp", telemetry.Timestamp.String(),
			"temperature", formatOptFloat(telemetry.Temperature, "°C"),
			"humidity", formatOptFloat(telemetry.Humidity, "%"),
			"pressure", formatOptFloat(telemetry.Pressure, "hPa"),
			"battery", formatOptFloat(telemetry.Battery, "V"),
			"run_count", formatOptUint32(telemetry.RunCount),
			"reading_id", formatOptUint32(telemetry.ReadingID),
		)

		err = repo.InsertReading(
			telemetry.StationID,
			telemetry.Timestamp,
			telemetry.Temperature,
			telemetry.Humidity,
			telemetry.Pressure,
			telemetry.Battery,
			telemetry.RunCount,
			telemetry.ReadingID,
		)

		if err != nil {
			slog.Error("failed to insert reading",
				"station_id", telemetry.StationID,
				"error", err,
			)
			return err
		}

		slog.Debug("successfully stored telemetry",
			"station_id", telemetry.StationID,
		)
		return nil
	})
}
