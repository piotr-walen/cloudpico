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

	// If true, prints all seen devices (can be noisy).
	Debug bool
}

// Listener wraps BlueZ scanning with context cancellation.
type Listener struct {
	adapter bluetooth.Adapter
	opts    Options
	log     *slog.Logger
}

func NewListener(opts Options, log *slog.Logger) *Listener {
	if opts.Adapter == "" {
		opts.Adapter = "hci0"
	}
	if log == nil {
		log = slog.Default()
	}

	return &Listener{
		adapter: *bluetooth.NewAdapter(opts.Adapter),
		opts:    opts,
		log:     log,
	}
}

// Run starts scanning and blocks until ctx is canceled or an error occurs.
// It calls onMatch for each matched observation.
// It does NOT stop on first match (you can do that in onMatch via cancel()).
func (l *Listener) Run(ctx context.Context, onMatch func(Match)) error {
	l.log.Info("ble: enabling adapter", "adapter", l.opts.Adapter)
	if err := l.adapter.Enable(); err != nil {
		return fmt.Errorf("ble enable (%s): %w", l.opts.Adapter, err)
	}
	l.log.Info("ble: adapter enabled", "adapter", l.opts.Adapter)

	// Stop scan on ctx cancel.
	go func() {
		<-ctx.Done()
		_ = l.adapter.StopScan()
	}()

	l.log.Info("ble: scanning started",
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

		if l.opts.Debug {
			l.log.Debug("ble: seen",
				"addr", obs.Address,
				"rssi", obs.RSSI,
				"name", obs.LocalName,
			)
		}

		// Local name filter (optional)
		if l.opts.Filter.LocalName != "" && obs.LocalName != l.opts.Filter.LocalName {
			return
		}

		for _, md := range r.ManufacturerData() {
			// CompanyID filter (optional if 0)
			if l.opts.Filter.CompanyID != 0 && md.CompanyID != l.opts.Filter.CompanyID {
				continue
			}
			// Prefix filter (optional)
			if !hasPrefix(md.Data, l.opts.Filter.ManufacturerDataPref) {
				continue
			}

			obs.CompanyID = md.CompanyID
			obs.Data = append([]byte(nil), md.Data...) // copy

			if onMatch != nil {
				onMatch(obs)
			}
			return
		}
	})

	// If ctx canceled, treat as clean shutdown.
	if ctx.Err() != nil {
		l.log.Info("ble: scanning stopped (context canceled)")
		return nil
	}

	if err != nil {
		return fmt.Errorf("ble scan: %w", err)
	}

	l.log.Info("ble: scanning stopped")
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
