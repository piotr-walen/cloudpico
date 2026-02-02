package httpapi

import (
	"cloudpico-server/pkg/config"
	"net/http"
)

func NewServer(config config.Config, mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:    config.HTTPAddr,
		Handler: requestLogger(mux),
	}
}
