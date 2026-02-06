package weather

import (
	"cloudpico-server/internal/modules/weather/controller"
	"cloudpico-server/internal/modules/weather/repository"
	"cloudpico-server/internal/modules/weather/service"
	"database/sql"
	"net/http"
)

func RegisterFeature(mux *http.ServeMux, db *sql.DB, subscriber MQTTSubscriber) {
	weatherRepository := repository.NewRepository(db)
	weatherService := service.NewService(weatherRepository)
	weatherService.Register(subscriber)
	weatherController := controller.NewWeatherController(weatherRepository)
	weatherController.RegisterRoutes(mux)

}
