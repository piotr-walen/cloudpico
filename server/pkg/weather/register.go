package weather

import (
	"cloudpico-server/pkg/weather/controller"
	"cloudpico-server/pkg/weather/repository"
	"database/sql"
	"net/http"
)

func RegisterFeature(mux *http.ServeMux, db *sql.DB) {
	weatherRepository := repository.NewRepository(db)
	weatherController := controller.NewWeatherController(weatherRepository)
	weatherController.RegisterRoutes(mux)
}
