-- Bucket readings in [from, to] into max_points buckets and aggregate by AVG per bucket.
-- Params: fromStr, maxPoints, toStr, fromStr, maxPoints, stationID, fromStr, toStr
SELECT
  CAST(sub.station_id AS TEXT) AS station_id,
  sub.bucket,
  COALESCE(AVG(sub.temperature_c), 0) AS value,
  COALESCE(AVG(sub.humidity_pct), 0) AS humidity_pct,
  COALESCE(AVG(sub.pressure_hpa), 0) AS pressure_hpa,
  COALESCE(AVG(sub.battery_v), 0) AS battery_v,
  AVG(sub.battery_soc_pct) AS battery_soc_pct
FROM (
  SELECT
    station_id,
    MIN(CAST(
      (julianday(ts) - julianday(?)) * ? / NULLIF(julianday(?) - julianday(?), 0)
    AS INT), ? - 1) AS bucket,
    temperature_c,
    humidity_pct,
    pressure_hpa,
    battery_v,
    battery_soc_pct
  FROM readings
  WHERE station_id = ? AND ts >= ? AND ts <= ?
) AS sub
GROUP BY sub.station_id, sub.bucket
ORDER BY sub.bucket ASC;
