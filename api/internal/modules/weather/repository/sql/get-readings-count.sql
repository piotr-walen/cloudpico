SELECT COUNT(*)
FROM readings
WHERE station_id = ? AND ts >= ? AND ts <= ?;
