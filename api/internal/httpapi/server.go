package httpapi

import (
	"cloudpico-api/internal/config"
	"net/http"
)

func NewServer(config config.Config, mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:    config.HTTPAddr,
		Handler: corsMiddleware(config.CORSAllowedOrigins, requestLogger(mux)),
	}
}
