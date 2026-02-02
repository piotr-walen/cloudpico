package repository

import (
	"database/sql"
	"time"

	"cloudpico-server/pkg/weather/types"
)

type WeatherRepository interface {
	GetStations() ([]types.Station, error)
	GetLatestReadings(stationID string) ([]types.Reading, error)
	GetReadings(stationID string, from time.Time, to time.Time, limit int) ([]types.Reading, error)
}

type repositoryImpl struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) WeatherRepository {
	return &repositoryImpl{db: db}
}

func (r *repositoryImpl) GetStations() ([]types.Station, error) {
	return []types.Station{
		{ID: "st-001", Name: "Central"},
		{ID: "st-002", Name: "North"},
	}, nil
}

func (r *repositoryImpl) GetLatestReadings(stationID string) ([]types.Reading, error) {
	return []types.Reading{
		{StationID: stationID, Time: time.Now(), Value: 12.34},
	}, nil
}

func (r *repositoryImpl) GetReadings(stationID string, from time.Time, to time.Time, limit int) ([]types.Reading, error) {
	return []types.Reading{
		{StationID: stationID, Time: time.Now(), Value: 12.34},
	}, nil
}
