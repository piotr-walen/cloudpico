package httpapi

import (
	"net/http"
)

func NewServer(addr string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /api/v1/stations", handleStations)
	mux.HandleFunc("GET /api/v1/stations/{id}/latest", handleLatest)
	mux.HandleFunc("GET /api/v1/stations/{id}/readings", handleReadings)

	return &http.Server{
		Addr:    addr,
		Handler: requestLogger(mux),
	}
}
