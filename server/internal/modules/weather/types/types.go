package types

import "time"

type Station struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Reading struct {
	StationID string    `json:"stationId"`
	Time      time.Time `json:"time"`
	Value     float64   `json:"value"`
}
