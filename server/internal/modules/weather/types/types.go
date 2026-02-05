package types

import "time"

type Station struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Reading struct {
	StationID   string    `json:"stationId"`
	Time        time.Time `json:"time"`
	Value       float64   `json:"value"`       // temperature °C
	HumidityPct float64   `json:"humidityPct"` // 0–100 or 0 if unset
	PressureHpa float64   `json:"pressureHpa"` // hPa or 0 if unset
}
