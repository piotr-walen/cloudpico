package sensor

import (
	"cloudpico-gateway/internal/config"
	"cloudpico-gateway/internal/mqtt"
	"context"
	"fmt"
	"time"

	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/host/v3"

	cloudpico_shared "cloudpico-shared/types"
)

func Run(ctx context.Context, cfg config.Config, mqttClient *mqtt.Client) error {

	sequence := 0
	if _, err := host.Init(); err != nil {
		return fmt.Errorf("host.Init: %w", err)
	}

	bus, err := i2creg.Open("") // default bus, usually /dev/i2c-1
	if err != nil {
		return fmt.Errorf("i2creg.Open: %w", err)
	}
	defer bus.Close()

	addr := cfg.BME280Address

	dev, err := bmxx80.NewI2C(bus, addr, &bmxx80.DefaultOpts)
	if err != nil {
		return fmt.Errorf("bmxx80.NewI2C: %w", err)
	}
	defer dev.Halt()

	ticker := time.NewTicker(cfg.SensorPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var env physic.Env
			if err := dev.Sense(&env); err != nil {
				return fmt.Errorf("Sense: %w", err)
			}

			temperature := env.Temperature.Celsius()

			// env.Humidity is a humidity level measurement stored as an int32 fixed
			// point integer at a precision of 0.00001%rH.
			// Valid values are between 0% and 100%.
			humidity := float64(env.Humidity) / 100_000.0 // convert to %

			// env.Pressure is a measurement of force applied to a surface per unit
			// area (stress) stored as an int64 nano Pascal.
			pressure := float64(env.Pressure) / 100_000_000_000.0 // convert to hPa

			sequence++

			telemetry := cloudpico_shared.Telemetry{
				StationID:   cfg.DeviceStationID,
				Timestamp:   time.Now(),
				Temperature: &temperature,
				Humidity:    &humidity,
				Pressure:    &pressure,
				Sequence:    &sequence,
			}

			if err := mqttClient.PublishTelemetry(telemetry); err != nil {
				return err
			}
		}
	}
}
