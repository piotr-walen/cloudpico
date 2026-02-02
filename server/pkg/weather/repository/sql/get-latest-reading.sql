SELECT CAST(station_id AS TEXT) AS station_id, ts, COALESCE(temperature_c, 0) AS value
FROM readings
WHERE station_id = ?
ORDER BY ts DESC
LIMIT 100;
