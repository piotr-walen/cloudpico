package service

import (
	"cloudpico-server/internal/modules/weather/repository"
	"cloudpico-server/internal/mqtt"
	"log/slog"
)

type Service struct {
	repository repository.WeatherRepository
}

func NewService(repository repository.WeatherRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Register(subscriber mqtt.MQTTSubscriber) {
	registerMQTTHandler(subscriber, s.repository, slog.Default())
}
