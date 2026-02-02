package repository

import (
	"database/sql"
	_ "embed"
	"log/slog"
	"time"

	"cloudpico-server/pkg/weather/types"
)

//go:embed sql/get-stations.sql
var getStationsSQL string

//go:embed sql/get-latest-reading.sql
var getLatestReadingSQL string

//go:embed sql/get-readings.sql
var getReadingsSQL string

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
	rows, err := r.db.Query(getStationsSQL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close stations rows", "error", err)
		}
	}()
	var out []types.Station
	for rows.Next() {
		var s types.Station
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *repositoryImpl) GetLatestReadings(stationID string) ([]types.Reading, error) {
	rows, err := r.db.Query(getLatestReadingSQL, stationID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close latest readings rows", "error", err)
		}
	}()
	return scanReadings(rows)
}

func (r *repositoryImpl) GetReadings(stationID string, from time.Time, to time.Time, limit int) ([]types.Reading, error) {
	fromStr := from.UTC().Format(time.RFC3339Nano)
	toStr := to.UTC().Format(time.RFC3339Nano)
	rows, err := r.db.Query(getReadingsSQL, stationID, fromStr, toStr, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("close readings rows", "error", err)
		}
	}()
	return scanReadings(rows)
}

func scanReadings(rows *sql.Rows) ([]types.Reading, error) {
	var out []types.Reading
	for rows.Next() {
		var rec types.Reading
		var ts string
		if err := rows.Scan(&rec.StationID, &ts, &rec.Value); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			t, _ = time.Parse(time.RFC3339, ts)
		}
		rec.Time = t
		out = append(out, rec)
	}
	return out, rows.Err()
}
