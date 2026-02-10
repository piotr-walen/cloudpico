package ble

import (
	"cloudpico-gateway/internal/mqtt"
	"cloudpico-gateway/internal/utils"
	"context"
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
	if h.seen[m.Address] == nil {
		h.seen[m.Address] = make(map[uint32]struct{})
	}
	if _, ok := h.seen[m.Address][sr.ReadingID]; ok {
		h.dedupMu.Unlock()
		return
	}
	h.seen[m.Address][sr.ReadingID] = struct{}{}
	if len(h.seen[m.Address]) > bleDedupMaxIDsPerDevice {
		h.seen[m.Address] = make(map[uint32]struct{})
		h.seen[m.Address][sr.ReadingID] = struct{}{}
	}
	h.dedupMu.Unlock()

	stationID := "outdoor"
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
		"reading_id", sr.ReadingID,
		"rssi", m.RSSI,
		"T", sr.Temperature, "P", sr.Pressure, "H", sr.Humidity,
		"data", utils.BytesToHex(m.Data),
	)
}

// StartListener starts the BLE listener with this handler.
func (h *BLESensorHandler) StartListener(ctx context.Context, listener *Listener) {
	go func() {
		err := listener.Run(ctx, h.HandleMatch)
		if err != nil {
			slog.Warn("ble listener could not be initialized; gateway continues without BLE",
				"error", err,
			)
		}
	}()
}
