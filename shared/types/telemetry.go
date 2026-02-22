package types

import "time"

// Telemetry represents a telemetry message from a weather station
type Telemetry struct {
	StationID   string    `json:"station_id"`
	Timestamp   time.Time `json:"timestamp"`
	Temperature *float64  `json:"temperature_c,omitempty"`
	Humidity    *float64  `json:"humidity_pct,omitempty"`
	Pressure    *float64  `json:"pressure_hpa,omitempty"`
	Battery     *float64  `json:"battery_v,omitempty"`
	BatterySOC  *float64  `json:"battery_soc_pct,omitempty"` // 0–100 from fuel gauge, or unset
	RunCount    *uint32   `json:"run_count,omitempty"`
	ReadingID   *uint32   `json:"reading_id,omitempty"`
}
