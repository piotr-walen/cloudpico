package ble

import (
	"cloudpico-gateway/internal/mqtt"
	"cloudpico-gateway/internal/utils"
	"fmt"
	"log/slog"
	"sync"
	"time"

	cloudpico_shared "cloudpico-shared/types"
)

// BLESensorHandler processes BLE sensor readings with deduplication and MQTT publishing.
type BLESensorHandler struct {
	mqttClient *mqtt.Client
	dedupMu    sync.Mutex
	seen       map[string]map[uint32]map[uint32]struct{}
}

// NewBLESensorHandler creates a new BLE sensor handler.
func NewBLESensorHandler(mqttClient *mqtt.Client) *BLESensorHandler {
	return &BLESensorHandler{
		mqttClient: mqttClient,
		seen:       make(map[string]map[uint32]map[uint32]struct{}),
	}
}

// HandleMatch processes a BLE match, deduplicates readings, and publishes telemetry.
func (h *BLESensorHandler) HandleMatch(m Match) {
	sr, err := ParseSensorPayload(m.Data)
	if err != nil {
		slog.Debug("ble: ignore non-sensor payload", "addr", m.Address, "error", err)
		return
	}

	h.dedupMu.Lock()
	deviceKey := fmt.Sprintf("%04X", sr.DeviceID)
	if h.seen[deviceKey] == nil {
		h.seen[deviceKey] = make(map[uint32]map[uint32]struct{})
	}
	if h.seen[deviceKey][sr.RunCount] == nil {
		h.seen[deviceKey][sr.RunCount] = make(map[uint32]struct{})
	}
	if _, ok := h.seen[deviceKey][sr.RunCount][sr.ReadingID]; ok {
		slog.Debug("ble: deduplicated reading", "addr", m.Address, "device_id", sr.DeviceID, "run_count", sr.RunCount, "reading_id", sr.ReadingID)
		h.dedupMu.Unlock()
		return
	}
	h.seen[deviceKey][sr.RunCount][sr.ReadingID] = struct{}{}
	h.dedupMu.Unlock()

	// Use device ID from payload as station ID (format: pico-{device_id})
	stationID := fmt.Sprintf("pico-%04X", sr.DeviceID)
	temp := sr.Temperature
	hum := sr.Humidity
	press := sr.Pressure
	battery := sr.Battery
	telemetry := cloudpico_shared.Telemetry{
		StationID:   stationID,
		Timestamp:   time.Now(),
		Temperature: &temp,
		Humidity:    &hum,
		Pressure:    &press,
		Battery:     &battery,
		RunCount:    &sr.RunCount,
		ReadingID:   &sr.ReadingID,
	}

	if err := h.mqttClient.PublishTelemetry(telemetry); err != nil {
		slog.Warn("ble: failed to publish telemetry", "addr", m.Address, "reading_id", sr.ReadingID, "error", err)
		return
	}
	slog.Info("ble: sensor reading published",
		"addr", m.Address,
		"device_id", sr.DeviceID,
		"station_id", stationID,
		"run_count", sr.RunCount,
		"reading_id", sr.ReadingID,
		"rssi", m.RSSI,
		"T", sr.Temperature, "P", sr.Pressure, "H", sr.Humidity, "V", sr.Battery,
		"data", utils.BytesToHex(m.Data),
	)
}
