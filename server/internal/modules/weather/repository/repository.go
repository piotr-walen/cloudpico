package repository

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"cloudpico-server/internal/modules/weather/types"
)

//go:embed sql/get-stations.sql
var getStationsSQL string

//go:embed sql/get-latest-reading.sql
var getLatestReadingSQL string

//go:embed sql/get-readings.sql
var getReadingsSQL string

//go:embed sql/get-readings-count.sql
var getReadingsCountSQL string

//go:embed sql/insert-reading.sql
var insertReadingSQL string

//go:embed sql/get-station-id-by-name.sql
var getStationIDByNameSQL string

type WeatherRepository interface {
	GetStations() ([]types.Station, error)
	GetLatestReadings(stationID string, limit int) ([]types.Reading, error)
	GetReadings(stationID string, from time.Time, to time.Time, limit int, offset int) ([]types.Reading, error)
	GetReadingsCount(stationID string, from time.Time, to time.Time) (int, error)
	InsertReading(stationID string, ts time.Time, temperature *float64, humidity *float64, pressure *float64) error
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

func (r *repositoryImpl) GetLatestReadings(stationID string, limit int) ([]types.Reading, error) {
	rows, err := r.db.Query(getLatestReadingSQL, stationID, limit)
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

func (r *repositoryImpl) GetReadings(stationID string, from time.Time, to time.Time, limit int, offset int) ([]types.Reading, error) {
	fromStr := from.UTC().Format(time.RFC3339Nano)
	toStr := to.UTC().Format(time.RFC3339Nano)
	rows, err := r.db.Query(getReadingsSQL, stationID, fromStr, toStr, limit, offset)
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

func (r *repositoryImpl) GetReadingsCount(stationID string, from time.Time, to time.Time) (int, error) {
	fromStr := from.UTC().Format(time.RFC3339Nano)
	toStr := to.UTC().Format(time.RFC3339Nano)
	var n int
	err := r.db.QueryRow(getReadingsCountSQL, stationID, fromStr, toStr).Scan(&n)
	return n, err
}

func scanReadings(rows *sql.Rows) ([]types.Reading, error) {
	var out []types.Reading
	for rows.Next() {
		var rec types.Reading
		var ts string
		if err := rows.Scan(&rec.StationID, &ts, &rec.Value, &rec.HumidityPct, &rec.PressureHpa); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			var err2 error
			t, err2 = time.Parse(time.RFC3339, ts)
			if err2 != nil {
				return nil, fmt.Errorf("parse timestamp %q: RFC3339Nano: %w; RFC3339: %w", ts, err, err2)
			}
		}
		rec.Time = t
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *repositoryImpl) InsertReading(stationID string, ts time.Time, temperature *float64, humidity *float64, pressure *float64) error {
	tsStr := ts.UTC().Format(time.RFC3339Nano)
	
	// Resolve station ID - stationID might be a name or an ID string
	// First try to parse as integer ID, if that fails, look up by name
	var dbStationID int
	var err error
	
	// Try parsing as integer first
	if parsedID, parseErr := strconv.Atoi(stationID); parseErr == nil {
		// It's a numeric ID, use it directly
		dbStationID = parsedID
	} else {
		// It's likely a station name, look it up
		err = r.db.QueryRow(getStationIDByNameSQL, stationID).Scan(&dbStationID)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("station not found: %q", stationID)
			}
			return fmt.Errorf("lookup station %q: %w", stationID, err)
		}
	}
	
	// Validate humidity range (0-100) if provided
	if humidity != nil {
		if *humidity < 0 || *humidity > 100 {
			return fmt.Errorf("humidity_pct out of range: %f (must be 0-100)", *humidity)
		}
	}
	
	// Validate pressure is positive if provided
	if pressure != nil {
		if *pressure <= 0 {
			return fmt.Errorf("pressure_hpa must be positive: %f", *pressure)
		}
	}
	
	var tempVal interface{}
	if temperature != nil {
		tempVal = *temperature
	}
	
	var humidityVal interface{}
	if humidity != nil {
		humidityVal = *humidity
	}
	
	var pressureVal interface{}
	if pressure != nil {
		pressureVal = *pressure
	}
	
	_, err = r.db.Exec(insertReadingSQL, dbStationID, tsStr, tempVal, humidityVal, pressureVal)
	if err != nil {
		return fmt.Errorf("insert reading: %w", err)
	}
	
	return nil
}
