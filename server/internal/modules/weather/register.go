package weather

import (
	"cloudpico-server/internal/modules/weather/controller"
	"cloudpico-server/internal/modules/weather/repository"
	"database/sql"
	"net/http"
)

func RegisterFeature(mux *http.ServeMux, db *sql.DB) {
	weatherRepository := repository.NewRepository(db)
	weatherController := controller.NewWeatherController(weatherRepository)
	weatherController.RegisterRoutes(mux)
}
