package service

import (
	"cloudpico-server/internal/modules/weather/repository"
	"cloudpico-server/internal/mqtt"
)

type Service struct {
	repository repository.WeatherRepository
}

func NewService(repository repository.WeatherRepository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Register(subscriber *mqtt.Subscriber) {
	registerMQTTHandler(subscriber, s.repository)
}
