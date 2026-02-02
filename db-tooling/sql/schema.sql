-- =========================
-- stations
-- =========================
CREATE TABLE IF NOT EXISTS stations (
  id         INTEGER PRIMARY KEY,                 -- rowid alias; fast PK
  name       TEXT    NOT NULL,
  created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  metadata   TEXT                                     -- optional; store JSON string if you want
);

-- Optional: prevent duplicate station names (comment out if you allow dupes)
CREATE UNIQUE INDEX IF NOT EXISTS idx_stations_name
ON stations(name);

-- =========================
-- readings
-- =========================
CREATE TABLE IF NOT EXISTS readings (
  station_id      INTEGER NOT NULL,
  ts              TEXT    NOT NULL,               -- ISO-8601 timestamp recommended
  temperature_c   REAL,
  humidity_pct    REAL,
  pressure_hpa    REAL,

  -- Composite primary key: one reading per station per timestamp
  PRIMARY KEY (station_id, ts),

  -- FK back to stations
  FOREIGN KEY (station_id) REFERENCES stations(id)
    ON UPDATE CASCADE
    ON DELETE CASCADE,

  -- Basic sanity checks (adjust ranges as you like)
  CHECK (humidity_pct IS NULL OR (humidity_pct >= 0.0 AND humidity_pct <= 100.0)),
  CHECK (pressure_hpa IS NULL OR pressure_hpa > 0.0)
);

-- Fast lookups by station and time range
CREATE INDEX IF NOT EXISTS idx_readings_station_ts
ON readings(station_id, ts);

-- Optional: time-based queries across all stations
CREATE INDEX IF NOT EXISTS idx_readings_ts
ON readings(ts);