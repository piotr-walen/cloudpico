package repository

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Minimal schema matching db-tooling/sql/schema.sql for in-memory tests.
const testSchema = `
CREATE TABLE IF NOT EXISTS stations (
  id         INTEGER PRIMARY KEY,
  name       TEXT    NOT NULL,
  created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  metadata   TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_stations_name ON stations(name);

CREATE TABLE IF NOT EXISTS readings (
  station_id      INTEGER NOT NULL,
  ts              TEXT    NOT NULL,
  temperature_c   REAL,
  humidity_pct    REAL,
  pressure_hpa    REAL,
  PRIMARY KEY (station_id, ts),
  FOREIGN KEY (station_id) REFERENCES stations(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_readings_station_ts ON readings(station_id, ts);
CREATE INDEX IF NOT EXISTS idx_readings_ts ON readings(ts);
`

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(testSchema); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
		t.Fatalf("exec schema: %v", err)
	}
	return db
}

func TestNewRepository(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	repo := NewRepository(db)
	if repo == nil {
		t.Fatal("NewRepository returned nil")
	}
}

func TestGetStations_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	repo := NewRepository(db)

	stations, err := repo.GetStations()
	if err != nil {
		t.Fatalf("GetStations: %v", err)
	}
	if len(stations) != 0 {
		t.Fatalf("GetStations: got %d stations, want 0", len(stations))
	}
}

func TestGetStations_WithData(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	_, err := db.Exec(`INSERT INTO stations (id, name) VALUES (1, 'Alpha'), (2, 'Beta')`)
	if err != nil {
		t.Fatalf("insert stations: %v", err)
	}
	repo := NewRepository(db)

	stations, err := repo.GetStations()
	if err != nil {
		t.Fatalf("GetStations: %v", err)
	}
	if len(stations) != 2 {
		t.Fatalf("GetStations: got %d stations, want 2", len(stations))
	}
	// Ordered by name: Alpha, Beta
	if stations[0].ID != "1" || stations[0].Name != "Alpha" {
		t.Errorf("first station: got id=%q name=%q, want id=1 name=Alpha", stations[0].ID, stations[0].Name)
	}
	if stations[1].ID != "2" || stations[1].Name != "Beta" {
		t.Errorf("second station: got id=%q name=%q, want id=2 name=Beta", stations[1].ID, stations[1].Name)
	}
}

func TestGetLatestReadings_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	_, err := db.Exec(`INSERT INTO stations (id, name) VALUES (1, 'Only')`)
	if err != nil {
		t.Fatalf("insert station: %v", err)
	}
	repo := NewRepository(db)

	readings, err := repo.GetLatestReadings("1")
	if err != nil {
		t.Fatalf("GetLatestReadings: %v", err)
	}
	if len(readings) != 0 {
		t.Fatalf("GetLatestReadings: got %d readings, want 0", len(readings))
	}
}

func TestGetLatestReadings_WithData(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	_, err := db.Exec(`INSERT INTO stations (id, name) VALUES (1, 'Central')`)
	if err != nil {
		t.Fatalf("insert station: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO readings (station_id, ts, temperature_c) VALUES
		(1, '2025-02-01T12:00:00Z', 10.0),
		(1, '2025-02-01T13:00:00Z', 11.5),
		(1, '2025-02-01T14:00:00Z', 12.0)
	`)
	if err != nil {
		t.Fatalf("insert readings: %v", err)
	}
	repo := NewRepository(db)

	readings, err := repo.GetLatestReadings("1")
	if err != nil {
		t.Fatalf("GetLatestReadings: %v", err)
	}
	if len(readings) != 3 {
		t.Fatalf("GetLatestReadings: got %d readings, want 3", len(readings))
	}
	// Order: newest first (14:00, 13:00, 12:00)
	if readings[0].Value != 12.0 || readings[1].Value != 11.5 || readings[2].Value != 10.0 {
		t.Errorf("GetLatestReadings order: got values %v, want [12, 11.5, 10]", []float64{readings[0].Value, readings[1].Value, readings[2].Value})
	}
	for i := range readings {
		if readings[i].StationID != "1" {
			t.Errorf("reading[%d].StationID = %q, want 1", i, readings[i].StationID)
		}
	}
}

func TestGetLatestReadings_UnknownStation(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	repo := NewRepository(db)

	readings, err := repo.GetLatestReadings("999")
	if err != nil {
		t.Fatalf("GetLatestReadings: %v", err)
	}
	if len(readings) != 0 {
		t.Fatalf("GetLatestReadings(999): got %d readings, want 0", len(readings))
	}
}

func TestGetReadings_EmptyRange(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	_, err := db.Exec(`INSERT INTO stations (id, name) VALUES (1, 'S1')`)
	if err != nil {
		t.Fatalf("insert station: %v", err)
	}
	repo := NewRepository(db)

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	readings, err := repo.GetReadings("1", from, to, 10)
	if err != nil {
		t.Fatalf("GetReadings: %v", err)
	}
	if len(readings) != 0 {
		t.Fatalf("GetReadings: got %d readings, want 0", len(readings))
	}
}

func TestGetReadings_WithData(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	_, err := db.Exec(`INSERT INTO stations (id, name) VALUES (1, 'S1')`)
	if err != nil {
		t.Fatalf("insert station: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO readings (station_id, ts, temperature_c) VALUES
		(1, '2025-02-01T10:00:00Z', 8.0),
		(1, '2025-02-01T11:00:00Z', 9.0),
		(1, '2025-02-01T12:00:00Z', 10.0),
		(1, '2025-02-01T13:00:00Z', 11.0),
		(1, '2025-02-01T14:00:00Z', 12.0)
	`)
	if err != nil {
		t.Fatalf("insert readings: %v", err)
	}
	repo := NewRepository(db)

	from := time.Date(2025, 2, 1, 11, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 1, 13, 59, 59, 0, time.UTC)
	readings, err := repo.GetReadings("1", from, to, 10)
	if err != nil {
		t.Fatalf("GetReadings: %v", err)
	}
	// 11:00, 12:00, 13:00 within range; order DESC so 13, 12, 11
	if len(readings) != 3 {
		t.Fatalf("GetReadings: got %d readings, want 3", len(readings))
	}
	if readings[0].Value != 11.0 || readings[1].Value != 10.0 || readings[2].Value != 9.0 {
		t.Errorf("GetReadings: got values %v, want [11, 10, 9]", []float64{readings[0].Value, readings[1].Value, readings[2].Value})
	}
}

func TestGetReadings_RespectsLimit(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	_, err := db.Exec(`INSERT INTO stations (id, name) VALUES (1, 'S1')`)
	if err != nil {
		t.Fatalf("insert station: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO readings (station_id, ts, temperature_c) VALUES
		(1, '2025-02-01T10:00:00Z', 10.0),
		(1, '2025-02-01T11:00:00Z', 11.0),
		(1, '2025-02-01T12:00:00Z', 12.0)
	`)
	if err != nil {
		t.Fatalf("insert readings: %v", err)
	}
	repo := NewRepository(db)

	from := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC)
	readings, err := repo.GetReadings("1", from, to, 2)
	if err != nil {
		t.Fatalf("GetReadings: %v", err)
	}
	if len(readings) != 2 {
		t.Fatalf("GetReadings(limit=2): got %d readings, want 2", len(readings))
	}
	// Newest first: 12, 11
	if readings[0].Value != 12.0 || readings[1].Value != 11.0 {
		t.Errorf("GetReadings limit: got values %v", []float64{readings[0].Value, readings[1].Value})
	}
}

func TestGetReadings_NullTemperature(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	_, err := db.Exec(`INSERT INTO stations (id, name) VALUES (1, 'S1')`)
	if err != nil {
		t.Fatalf("insert station: %v", err)
	}
	_, err = db.Exec(`INSERT INTO readings (station_id, ts, temperature_c) VALUES (1, '2025-02-01T12:00:00Z', NULL)`)
	if err != nil {
		t.Fatalf("insert reading: %v", err)
	}
	repo := NewRepository(db)

	from := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC)
	readings, err := repo.GetReadings("1", from, to, 10)
	if err != nil {
		t.Fatalf("GetReadings: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("GetReadings: got %d readings, want 1", len(readings))
	}
	// COALESCE(temperature_c, 0) in SQL
	if readings[0].Value != 0 {
		t.Errorf("null temperature_c: got value %v, want 0", readings[0].Value)
	}
}

// Ensure repo implements the interface.
var _ WeatherRepository = (*repositoryImpl)(nil)

func TestRepository_ImplementsInterface(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	repo := NewRepository(db)
	// Compile-time check; also call all methods for coverage.
	_, _ = repo.GetStations()
	_, _ = repo.GetLatestReadings("1")
	_, _ = repo.GetReadings("1", time.Now().Add(-24*time.Hour), time.Now(), 10)
}
