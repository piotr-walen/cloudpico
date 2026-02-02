PRAGMA foreign_keys = ON;

BEGIN;

-- Insert stations if they don't exist yet
INSERT OR IGNORE INTO stations (name, metadata) VALUES
  ('Warsaw-Center',  '{"lat":52.2297,"lon":21.0122,"elev_m":110}'),
  ('Gdansk-Port',    '{"lat":54.3520,"lon":18.6466,"elev_m":5}'),
  ('Krakow-OldTown', '{"lat":50.0647,"lon":19.9450,"elev_m":219}');

-- Update metadata in case station rows already existed
UPDATE stations
SET metadata = '{"lat":52.2297,"lon":21.0122,"elev_m":110}'
WHERE name = 'Warsaw-Center';

UPDATE stations
SET metadata = '{"lat":54.3520,"lon":18.6466,"elev_m":5}'
WHERE name = 'Gdansk-Port';

UPDATE stations
SET metadata = '{"lat":50.0647,"lon":19.9450,"elev_m":219}'
WHERE name = 'Krakow-OldTown';

-- Generate last 24 hours of hourly readings (random-ish, no trig functions)
WITH RECURSIVE
  hours(n) AS (
    SELECT 0
    UNION ALL
    SELECT n + 1 FROM hours WHERE n < 23
  ),
  station_ids AS (
    SELECT id AS station_id, name
    FROM stations
    WHERE name IN ('Warsaw-Center','Gdansk-Port','Krakow-OldTown')
  ),
  ts_gen AS (
    SELECT
      s.station_id,
      s.name,
      strftime('%Y-%m-%dT%H:%M:%SZ',
        datetime('now', 'utc', printf('-%d hours', 23 - h.n))
      ) AS ts
    FROM station_ids s
    CROSS JOIN hours h
  )
INSERT OR REPLACE INTO readings (station_id, ts, temperature_c, humidity_pct, pressure_hpa)
SELECT
  station_id,
  ts,

  -- temperature base per station + random noise
  ROUND(
    (CASE name
      WHEN 'Warsaw-Center'  THEN 6.0
      WHEN 'Gdansk-Port'    THEN 5.0
      WHEN 'Krakow-OldTown' THEN 6.5
      ELSE 6.0
     END)
    + ((abs(random()) % 800) / 100.0 - 4.0)   -- [-4.0, +4.0)
  , 1) AS temperature_c,

  -- humidity 40..95
  ROUND(40.0 + (abs(random()) % 5500) / 100.0, 1) AS humidity_pct,

  -- pressure 1000..1030
  ROUND(1000.0 + (abs(random()) % 3000) / 100.0, 1) AS pressure_hpa
FROM ts_gen;

COMMIT;
