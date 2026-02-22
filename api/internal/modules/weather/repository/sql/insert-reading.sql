INSERT OR REPLACE INTO readings (station_id, ts, temperature_c, humidity_pct, pressure_hpa, battery_v, battery_soc_pct, run_count, reading_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
