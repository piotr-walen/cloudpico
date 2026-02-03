package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv   string
	LogLevel slog.Level
	HTTPAddr string

	// StaticDir is the absolute path to the directory served at /static/.
	// Set via STATIC_DIR (relative paths are resolved against the process working directory at startup).
	StaticDir string

	SQLiteDriver          string
	SQLiteDSN             string
	SQLitePath            string
	SQLiteMaxOpenConns    int
	SQLiteMaxIdleConns    int
	SQLiteConnMaxLifetime time.Duration
}

func LoadFromEnv() (Config, error) {
	appEnv := strings.TrimSpace(os.Getenv("APP_ENV"))
	if appEnv == "" {
		appEnv = "dev"
	}
	switch appEnv {
	case "dev", "prod":
	default:
		return Config{}, fmt.Errorf("invalid APP_ENV %q (allowed: dev, prod)", appEnv)
	}

	logLevelStr := strings.TrimSpace(os.Getenv("LOG_LEVEL"))
	if logLevelStr == "" {
		logLevelStr = "info"
	}
	level, err := parseLogLevel(logLevelStr)
	if err != nil {
		return Config{}, err
	}

	httpAddr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	staticDir := strings.TrimSpace(os.Getenv("STATIC_DIR"))
	if staticDir == "" {
		staticDir = "static"
	}
	staticDir, err = filepath.Abs(staticDir)
	if err != nil {
		return Config{}, fmt.Errorf("STATIC_DIR %q: %w", staticDir, err)
	}

	sqliteDriver := strings.TrimSpace(os.Getenv("SQLITE_DRIVER"))
	if sqliteDriver == "" {
		sqliteDriver = "sqlite3"
	}
	sqliteDSN := strings.TrimSpace(os.Getenv("SQLITE_DSN"))
	sqlitePath := strings.TrimSpace(os.Getenv("SQLITE_PATH"))
	if sqlitePath == "" {
		sqlitePath = "../dev/sqlite/app.db"
	}

	sqliteMaxOpenConnsStr := strings.TrimSpace(os.Getenv("SQLITE_MAX_OPEN_CONNS"))
	if sqliteMaxOpenConnsStr == "" {
		sqliteMaxOpenConnsStr = "1"
	}
	sqliteMaxOpenConns, err := strconv.Atoi(sqliteMaxOpenConnsStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid SQLITE_MAX_OPEN_CONNS %q: %w", sqliteMaxOpenConnsStr, err)
	}

	sqliteMaxIdleConnsStr := strings.TrimSpace(os.Getenv("SQLITE_MAX_IDLE_CONNS"))
	if sqliteMaxIdleConnsStr == "" {
		sqliteMaxIdleConnsStr = "1"
	}
	sqliteMaxIdleConns, err := strconv.Atoi(sqliteMaxIdleConnsStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid SQLITE_MAX_IDLE_CONNS %q: %w", sqliteMaxIdleConnsStr, err)
	}

	sqliteConnMaxLifetimeStr := strings.TrimSpace(os.Getenv("SQLITE_CONN_MAX_LIFETIME"))
	if sqliteConnMaxLifetimeStr == "" {
		sqliteConnMaxLifetimeStr = "0s"
	}
	sqliteConnMaxLifetime, err := time.ParseDuration(sqliteConnMaxLifetimeStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid SQLITE_CONN_MAX_LIFETIME %q: %w", strings.TrimSpace(os.Getenv("SQLITE_CONN_MAX_LIFETIME")), err)
	}

	return Config{
		AppEnv:                appEnv,
		LogLevel:              level,
		HTTPAddr:              httpAddr,
		StaticDir:             staticDir,
		SQLiteDriver:          sqliteDriver,
		SQLiteDSN:             sqliteDSN,
		SQLitePath:            sqlitePath,
		SQLiteMaxOpenConns:    sqliteMaxOpenConns,
		SQLiteMaxIdleConns:    sqliteMaxIdleConns,
		SQLiteConnMaxLifetime: sqliteConnMaxLifetime,
	}, nil
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid LOG_LEVEL %q (allowed: debug, info, warn, error)", s)
	}
}
