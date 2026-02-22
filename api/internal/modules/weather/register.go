package weather

import (
	"cloudpico-api/internal/modules/weather/controller"
	"cloudpico-api/internal/modules/weather/repository"
	"cloudpico-api/internal/modules/weather/service"
	"cloudpico-api/internal/mqtt"
	"database/sql"
	"net/http"
)

func RegisterFeature(mux *http.ServeMux, db *sql.DB, subscriber *mqtt.Subscriber) {
	weatherRepository := repository.NewRepository(db)
	weatherService := service.NewService(weatherRepository)
	weatherService.Register(subscriber)
	weatherController := controller.NewWeatherController(weatherRepository)
	weatherController.RegisterRoutes(mux)

}
