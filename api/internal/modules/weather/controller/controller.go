package controller

import (
	"cloudpico-api/internal/modules/weather/repository"
	"net/http"
)

type WeatherController interface {
	RegisterRoutes(mux *http.ServeMux)
}

type weatherControllerImpl struct {
	repository repository.WeatherRepository
}

func NewWeatherController(repository repository.WeatherRepository) WeatherController {
	return &weatherControllerImpl{repository: repository}
}

func (c *weatherControllerImpl) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /stations", c.handleStations)
	mux.HandleFunc("GET /stations/with-latest", c.handleStationsWithLatest)
	mux.HandleFunc("GET /stations/{id}/latest", c.handleLatest)
	mux.HandleFunc("GET /stations/{id}/readings", c.handleReadings)
	mux.HandleFunc("GET /stations/{id}/history/readings", c.handleHistoryReadings)
	mux.HandleFunc("GET /stations/{id}/history/chart", c.handleHistoryChart)
	mux.HandleFunc("GET /stations/{id}/history/count", c.handleHistoryCount)
}
