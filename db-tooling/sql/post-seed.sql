SELECT 'stations' AS table_name, COUNT(*) AS rows FROM stations
UNION ALL
SELECT 'readings' AS table_name, COUNT(*) AS rows FROM readings;
