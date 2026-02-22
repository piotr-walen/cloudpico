SELECT CAST(station_id AS TEXT) AS station_id, ts,
  COALESCE(temperature_c, 0) AS value,
  COALESCE(humidity_pct, 0) AS humidity_pct,
  COALESCE(pressure_hpa, 0) AS pressure_hpa,
  COALESCE(battery_v, 0) AS battery_v,
  battery_soc_pct AS battery_soc_pct
FROM readings
WHERE station_id = ? AND ts >= ? AND ts <= ?
ORDER BY ts DESC
LIMIT ? OFFSET ?;
