package httpapi

import (
	"cloudpico-server/pkg/config"
	"net/http"
)

func NewServer(config config.Config) *http.Server {
	mux := http.NewServeMux()

	weatherAPI := NewWeatherAPI(config)

	mux.HandleFunc("GET /healthz", weatherAPI.HandleHealthz)
	mux.HandleFunc("GET /api/v1/stations", weatherAPI.HandleStations)
	mux.HandleFunc("GET /api/v1/stations/{id}/latest", weatherAPI.HandleLatest)
	mux.HandleFunc("GET /api/v1/stations/{id}/readings", weatherAPI.HandleReadings)

	return &http.Server{
		Addr:    config.HTTPAddr,
		Handler: requestLogger(mux),
	}
}
