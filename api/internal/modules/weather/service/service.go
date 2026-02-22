package service

import (
	"cloudpico-api/internal/modules/weather/repository"
	"cloudpico-api/internal/mqtt"
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
