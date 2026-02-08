package repository

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Minimal schema matching tools/migrate/sql/0001_schema.sql for in-memory tests.
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

	readings, err := repo.GetLatestReadings("1", 100)
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

	readings, err := repo.GetLatestReadings("1", 100)
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

func TestGetLatestReadings_RespectsLimit(t *testing.T) {
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
		(1, '2025-02-01T14:00:00Z', 12.0),
		(1, '2025-02-01T15:00:00Z', 13.0),
		(1, '2025-02-01T16:00:00Z', 14.0)
	`)
	if err != nil {
		t.Fatalf("insert readings: %v", err)
	}
	repo := NewRepository(db)

	readings, err := repo.GetLatestReadings("1", 2)
	if err != nil {
		t.Fatalf("GetLatestReadings: %v", err)
	}
	if len(readings) != 2 {
		t.Fatalf("GetLatestReadings(limit=2): got %d readings, want 2", len(readings))
	}
	// Newest first: 16:00 (14.0), 15:00 (13.0)
	if readings[0].Value != 14.0 || readings[1].Value != 13.0 {
		t.Errorf("GetLatestReadings order: got values %v, want [14, 13]", []float64{readings[0].Value, readings[1].Value})
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

	readings, err := repo.GetLatestReadings("999", 100)
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
	readings, err := repo.GetReadings("1", from, to, 10, 0)
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
	readings, err := repo.GetReadings("1", from, to, 10, 0)
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

func TestGetReadings_HumidityAndPressure(t *testing.T) {
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
	// Insert readings with mixed humidity/pressure: set values and NULLs (COALESCE → 0)
	_, err = db.Exec(`
		INSERT INTO readings (station_id, ts, temperature_c, humidity_pct, pressure_hpa) VALUES
		(1, '2025-02-01T10:00:00Z', 8.0, 65.0, 1013.25),
		(1, '2025-02-01T11:00:00Z', 9.0, NULL, 1012.0),
		(1, '2025-02-01T12:00:00Z', 10.0, 70.5, NULL),
		(1, '2025-02-01T13:00:00Z', 11.0, NULL, NULL)
	`)
	if err != nil {
		t.Fatalf("insert readings: %v", err)
	}
	repo := NewRepository(db)

	from := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC)
	readings, err := repo.GetReadings("1", from, to, 10, 0)
	if err != nil {
		t.Fatalf("GetReadings: %v", err)
	}
	// Order DESC: 13:00, 12:00, 11:00, 10:00
	if len(readings) != 4 {
		t.Fatalf("GetReadings: got %d readings, want 4", len(readings))
	}
	// 13:00 — both NULL → COALESCE to 0
	if readings[0].HumidityPct != 0 || readings[0].PressureHpa != 0 {
		t.Errorf("reading 13:00 (NULL/NULL): got HumidityPct=%v PressureHpa=%v, want 0, 0", readings[0].HumidityPct, readings[0].PressureHpa)
	}
	// 12:00 — humidity set, pressure NULL
	if readings[1].HumidityPct != 70.5 || readings[1].PressureHpa != 0 {
		t.Errorf("reading 12:00 (70.5/NULL): got HumidityPct=%v PressureHpa=%v, want 70.5, 0", readings[1].HumidityPct, readings[1].PressureHpa)
	}
	// 11:00 — humidity NULL, pressure set
	if readings[2].HumidityPct != 0 || readings[2].PressureHpa != 1012.0 {
		t.Errorf("reading 11:00 (NULL/1012): got HumidityPct=%v PressureHpa=%v, want 0, 1012", readings[2].HumidityPct, readings[2].PressureHpa)
	}
	// 10:00 — both set
	if readings[3].HumidityPct != 65.0 || readings[3].PressureHpa != 1013.25 {
		t.Errorf("reading 10:00 (65/1013.25): got HumidityPct=%v PressureHpa=%v, want 65, 1013.25", readings[3].HumidityPct, readings[3].PressureHpa)
	}
	// Temperature still correct
	if readings[0].Value != 11.0 || readings[3].Value != 8.0 {
		t.Errorf("temperature: got [0]=%v [3]=%v, want 11, 8", readings[0].Value, readings[3].Value)
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
	readings, err := repo.GetReadings("1", from, to, 2, 0)
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

func TestGetReadings_RespectsOffset(t *testing.T) {
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
		(1, '2025-02-01T12:00:00Z', 12.0),
		(1, '2025-02-01T13:00:00Z', 13.0)
	`)
	if err != nil {
		t.Fatalf("insert readings: %v", err)
	}
	repo := NewRepository(db)

	from := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC)
	readings, err := repo.GetReadings("1", from, to, 2, 2)
	if err != nil {
		t.Fatalf("GetReadings: %v", err)
	}
	if len(readings) != 2 {
		t.Fatalf("GetReadings(limit=2, offset=2): got %d readings, want 2", len(readings))
	}
	// Order DESC: 13, 12, 11, 10. Offset 2 gives 11, 10
	if readings[0].Value != 11.0 || readings[1].Value != 10.0 {
		t.Errorf("GetReadings offset: got values %v, want [11, 10]", []float64{readings[0].Value, readings[1].Value})
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
	readings, err := repo.GetReadings("1", from, to, 10, 0)
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

func TestGetReadingsCount(t *testing.T) {
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
	n, err := repo.GetReadingsCount("1", from, to)
	if err != nil {
		t.Fatalf("GetReadingsCount: %v", err)
	}
	if n != 3 {
		t.Errorf("GetReadingsCount: got %d, want 3", n)
	}
	n, err = repo.GetReadingsCount("1", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetReadingsCount (empty range): %v", err)
	}
	if n != 0 {
		t.Errorf("GetReadingsCount empty range: got %d, want 0", n)
	}
}

func TestInsertReading_ByNumericStationID(t *testing.T) {
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
	repo := NewRepository(db)

	ts := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	temp := 22.5
	hum := 65.0
	press := 1013.25

	err = repo.InsertReading("1", ts, &temp, &hum, &press)
	if err != nil {
		t.Fatalf("InsertReading: %v", err)
	}

	readings, err := repo.GetLatestReadings("1", 1)
	if err != nil {
		t.Fatalf("GetLatestReadings: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("GetLatestReadings: got %d readings, want 1", len(readings))
	}
	if readings[0].Value != 22.5 || readings[0].HumidityPct != 65.0 || readings[0].PressureHpa != 1013.25 {
		t.Errorf("reading: got temp=%v humidity=%v pressure=%v, want 22.5, 65, 1013.25",
			readings[0].Value, readings[0].HumidityPct, readings[0].PressureHpa)
	}
	if readings[0].StationID != "1" {
		t.Errorf("StationID: got %q, want 1", readings[0].StationID)
	}
}

func TestInsertReading_ByStationName(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("close db: %v", closeErr)
		}
	}()
	_, err := db.Exec(`INSERT INTO stations (id, name) VALUES (2, 'Alpha')`)
	if err != nil {
		t.Fatalf("insert station: %v", err)
	}
	repo := NewRepository(db)

	ts := time.Date(2025, 3, 15, 10, 30, 0, 0, time.UTC)
	temp := 18.0
	hum := 50.0
	press := 1015.0

	err = repo.InsertReading("Alpha", ts, &temp, &hum, &press)
	if err != nil {
		t.Fatalf("InsertReading(Alpha): %v", err)
	}

	readings, err := repo.GetLatestReadings("2", 1)
	if err != nil {
		t.Fatalf("GetLatestReadings: %v", err)
	}
	if len(readings) != 1 {
		t.Fatalf("GetLatestReadings: got %d readings, want 1", len(readings))
	}
	if readings[0].Value != 18.0 || readings[0].HumidityPct != 50.0 || readings[0].PressureHpa != 1015.0 {
		t.Errorf("reading: got temp=%v humidity=%v pressure=%v, want 18, 50, 1015",
			readings[0].Value, readings[0].HumidityPct, readings[0].PressureHpa)
	}
}

func TestInsertReading_InvalidHumidity(t *testing.T) {
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

	ts := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	temp := 20.0

	t.Run("humidity_below_zero", func(t *testing.T) {
		hum := -1.0
		press := 1013.0
		err := repo.InsertReading("1", ts, &temp, &hum, &press)
		if err == nil {
			t.Fatal("InsertReading: expected error for humidity -1")
		}
		if !strings.Contains(err.Error(), "humidity_pct") || !strings.Contains(err.Error(), "0-100") {
			t.Errorf("error message: got %q", err.Error())
		}
	})

	t.Run("humidity_above_100", func(t *testing.T) {
		hum := 101.0
		press := 1013.0
		err := repo.InsertReading("1", ts, &temp, &hum, &press)
		if err == nil {
			t.Fatal("InsertReading: expected error for humidity 101")
		}
		if !strings.Contains(err.Error(), "humidity_pct") || !strings.Contains(err.Error(), "0-100") {
			t.Errorf("error message: got %q", err.Error())
		}
	})
}

func TestInsertReading_InvalidPressure(t *testing.T) {
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

	ts := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	temp := 20.0
	hum := 50.0

	t.Run("pressure_zero", func(t *testing.T) {
		press := 0.0
		err := repo.InsertReading("1", ts, &temp, &hum, &press)
		if err == nil {
			t.Fatal("InsertReading: expected error for pressure 0")
		}
		if !strings.Contains(err.Error(), "pressure_hpa") || !strings.Contains(err.Error(), "positive") {
			t.Errorf("error message: got %q", err.Error())
		}
	})

	t.Run("pressure_negative", func(t *testing.T) {
		press := -10.0
		err := repo.InsertReading("1", ts, &temp, &hum, &press)
		if err == nil {
			t.Fatal("InsertReading: expected error for pressure -10")
		}
		if !strings.Contains(err.Error(), "pressure_hpa") || !strings.Contains(err.Error(), "positive") {
			t.Errorf("error message: got %q", err.Error())
		}
	})
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
	_, _ = db.Exec(`INSERT INTO stations (id, name) VALUES (1, 'S1')`)
	repo := NewRepository(db)
	// Compile-time check; also call all methods for coverage.
	_, _ = repo.GetStations()
	_, _ = repo.GetLatestReadings("1", 100)
	_, _ = repo.GetReadings("1", time.Now().Add(-24*time.Hour), time.Now(), 10, 0)
	_, _ = repo.GetReadingsCount("1", time.Now().Add(-24*time.Hour), time.Now())
	temp, hum, press := 20.0, 50.0, 1013.0
	_ = repo.InsertReading("1", time.Now(), &temp, &hum, &press)
}
