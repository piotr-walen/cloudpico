package controller

import (
	"cloudpico-server/pkg/weather/repository"
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
	mux.HandleFunc("GET /api/v1/stations", c.handleStations)
	mux.HandleFunc("GET /api/v1/stations/{id}/latest", c.handleLatest)
	mux.HandleFunc("GET /api/v1/stations/{id}/readings", c.handleReadings)
}
