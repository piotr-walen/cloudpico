package ble

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"tinygo.org/x/bluetooth"
)

// Match is a single observation of your Pico beacon.
type Match struct {
	Address   string
	RSSI      int16
	LocalName string
	CompanyID uint16
	Data      []byte
	SeenAt    time.Time
}

type Filter struct {
	LocalName            string
	CompanyID            uint16
	ManufacturerDataPref []byte
}

type Options struct {
	Adapter string // "hci0" by default
	Filter  Filter
}

// Listener wraps BlueZ scanning with context cancellation.
type Listener struct {
	adapter *bluetooth.Adapter
	opts    Options
}

func NewListener(opts Options) *Listener {
	if opts.Adapter == "" {
		opts.Adapter = "hci0"
	}

	return &Listener{
		adapter: bluetooth.NewAdapter(opts.Adapter),
		opts:    opts,
	}
}

func (l *Listener) Run(ctx context.Context, onMatch func(Match)) error {
	slog.Info("ble: enabling adapter", "adapter", l.opts.Adapter)
	if err := l.adapter.Enable(); err != nil {
		return fmt.Errorf("ble enable (%s): %w", l.opts.Adapter, err)
	}
	slog.Info("ble: adapter enabled", "adapter", l.opts.Adapter)

	go func() {
		<-ctx.Done()
		_ = l.adapter.StopScan()
	}()

	slog.Info("ble: scanning started",
		"filter_name", l.opts.Filter.LocalName,
		"filter_company", fmt.Sprintf("0x%04X", l.opts.Filter.CompanyID),
		"filter_prefix", fmt.Sprintf("% X", l.opts.Filter.ManufacturerDataPref),
	)

	// adapter.Scan blocks until StopScan() or error.
	err := l.adapter.Scan(func(a *bluetooth.Adapter, r bluetooth.ScanResult) {
		obs := Match{
			Address:   r.Address.String(),
			RSSI:      r.RSSI,
			LocalName: r.LocalName(),
			SeenAt:    time.Now(),
		}

		// Collect manufacturer data for debug logging
		var allMfgData []struct {
			CompanyID uint16
			Data      []byte
		}
		for _, md := range r.ManufacturerData() {
			allMfgData = append(allMfgData, struct {
				CompanyID uint16
				Data      []byte
			}{md.CompanyID, append([]byte(nil), md.Data...)})
		}

		if l.opts.Filter.LocalName != "" && obs.LocalName != l.opts.Filter.LocalName {
			return
		}

		for _, md := range r.ManufacturerData() {
			if l.opts.Filter.CompanyID != 0 && md.CompanyID != l.opts.Filter.CompanyID {
				continue
			}
			
			// Debug: log raw manufacturer data
			slog.Debug("ble: raw manufacturer data",
				"company_id", fmt.Sprintf("0x%04X", md.CompanyID),
				"data_len", len(md.Data),
				"data_hex", fmt.Sprintf("% X", md.Data),
			)
			
			// Extract payload: Company ID may or may not be included by the BLE library
			// Expected formats:
			// 1. Company ID already stripped: payload starts with magic (24 bytes)
			// 2. Company ID included: Company ID (2 bytes) + payload (24 bytes) = 26 bytes total
			var payload []byte
			if len(md.Data) >= 2 && hasPrefix(md.Data, l.opts.Filter.ManufacturerDataPref) {
				// Magic bytes at start - Company ID already stripped
				payload = md.Data
				slog.Debug("ble: magic at start, using data as-is", "payload_len", len(payload))
			} else if len(md.Data) >= 26 && hasPrefix(md.Data[2:], l.opts.Filter.ManufacturerDataPref) {
				// Magic bytes at offset 2 - Company ID included, strip it
				payload = md.Data[2:]
				slog.Debug("ble: magic at offset 2, stripped Company ID", "payload_len", len(payload))
			} else {
				// No match, skip
				dataStart := "N/A"
				if len(md.Data) >= 2 {
					dataStart = fmt.Sprintf("%02X %02X", md.Data[0], md.Data[1])
				}
				slog.Debug("ble: no magic match",
					"data_len", len(md.Data),
					"data_start", dataStart,
					"data_at_2", func() string {
						if len(md.Data) >= 4 {
							return fmt.Sprintf("%02X %02X", md.Data[2], md.Data[3])
						}
						return "N/A"
					}(),
				)
				continue
			}
			
			// Validate payload length
			if len(payload) < 24 {
				slog.Debug("ble: payload too short after extraction",
					"payload_len", len(payload),
					"expected", 24,
				)
				continue
			}
			
			// Trim to exactly 24 bytes if longer (shouldn't happen, but be safe)
			if len(payload) > 24 {
				slog.Debug("ble: payload longer than expected, trimming",
					"payload_len", len(payload),
					"expected", 24,
				)
				payload = payload[:24]
			}

			obs.CompanyID = md.CompanyID
			obs.Data = append([]byte(nil), payload...)
			
			slog.Debug("ble: extracted payload",
				"payload_len", len(obs.Data),
				"payload_hex", fmt.Sprintf("% X", obs.Data),
			)

			if onMatch != nil {
				onMatch(obs)
			}
			return
		}
	})

	// If ctx canceled, treat as clean shutdown.
	if ctx.Err() != nil {
		slog.Info("ble: scanning stopped (context canceled)")
		return nil
	}

	if err != nil {
		return fmt.Errorf("ble scan: %w", err)
	}

	slog.Info("ble: scanning stopped")
	return nil
}

func hasPrefix(b, pref []byte) bool {
	if len(pref) == 0 {
		return true
	}
	if len(b) < len(pref) {
		return false
	}
	for i := range pref {
		if b[i] != pref[i] {
			return false
		}
	}
	return true
}
