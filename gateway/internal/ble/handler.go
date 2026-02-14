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

const bleDedupMaxIDsPerDevice = 500

// BLESensorHandler processes BLE sensor readings with deduplication and MQTT publishing.
type BLESensorHandler struct {
	mqttClient *mqtt.Client
	dedupMu    sync.Mutex
	seen       map[string]map[uint32]struct{}
}

// NewBLESensorHandler creates a new BLE sensor handler.
func NewBLESensorHandler(mqttClient *mqtt.Client) *BLESensorHandler {
	return &BLESensorHandler{
		mqttClient: mqttClient,
		seen:       make(map[string]map[uint32]struct{}),
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
	deviceKey := fmt.Sprintf("%08X", sr.DeviceID)
	if h.seen[deviceKey] == nil {
		h.seen[deviceKey] = make(map[uint32]struct{})
	}
	if _, ok := h.seen[deviceKey][sr.ReadingID]; ok {
		h.dedupMu.Unlock()
		return
	}
	h.seen[deviceKey][sr.ReadingID] = struct{}{}
	if len(h.seen[deviceKey]) > bleDedupMaxIDsPerDevice {
		h.seen[deviceKey] = make(map[uint32]struct{})
		h.seen[deviceKey][sr.ReadingID] = struct{}{}
	}
	h.dedupMu.Unlock()

	// Use device ID from payload as station ID (format: pico-{device_id})
	stationID := fmt.Sprintf("pico-%08X", sr.DeviceID)
	temp := sr.Temperature
	hum := sr.Humidity
	press := sr.Pressure
	seq := int(sr.ReadingID)
	telemetry := cloudpico_shared.Telemetry{
		StationID:   stationID,
		Timestamp:   time.Now(),
		Temperature: &temp,
		Humidity:    &hum,
		Pressure:    &press,
		Sequence:    &seq,
	}

	if err := h.mqttClient.PublishTelemetry(telemetry); err != nil {
		slog.Warn("ble: failed to publish telemetry", "addr", m.Address, "reading_id", sr.ReadingID, "error", err)
		return
	}
	slog.Info("ble: sensor reading published",
		"addr", m.Address,
		"device_id", sr.DeviceID,
		"station_id", stationID,
		"reading_id", sr.ReadingID,
		"rssi", m.RSSI,
		"T", sr.Temperature, "P", sr.Pressure, "H", sr.Humidity,
		"data", utils.BytesToHex(m.Data),
	)
}
