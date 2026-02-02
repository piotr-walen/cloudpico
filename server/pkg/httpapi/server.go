package httpapi

import (
	"net/http"
	"time"
)

func NewServer(addr string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /api/v1/stations", handleStations)
	mux.HandleFunc("GET /api/v1/stations/{id}/latest", handleLatest)
	mux.HandleFunc("GET /api/v1/stations/{id}/readings", handleReadings)

	return &http.Server{
		Addr:              addr,
		Handler:           requestLogger(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
}
