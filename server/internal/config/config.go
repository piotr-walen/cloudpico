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

	Driver          string
	DSN             string
	Path            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
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

	driver := strings.TrimSpace(os.Getenv("DB_DRIVER"))
	if driver == "" {
		driver = "sqlite3"
	}
	dsn := strings.TrimSpace(os.Getenv("DB_DSN"))
	path := strings.TrimSpace(os.Getenv("SQLITE_PATH"))
	if path == "" {
		path = "../dev/sqlite/app.db"
	}

	maxOpenConnsStr := strings.TrimSpace(os.Getenv("DB_MAX_OPEN_CONNS"))
	if maxOpenConnsStr == "" {
		maxOpenConnsStr = "1"
	}
	maxOpenConns, err := strconv.Atoi(maxOpenConnsStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DB_MAX_OPEN_CONNS %q: %w", maxOpenConnsStr, err)
	}

	maxIdleConnsStr := strings.TrimSpace(os.Getenv("DB_MAX_IDLE_CONNS"))
	if maxIdleConnsStr == "" {
		maxIdleConnsStr = "1"
	}
	maxIdleConns, err := strconv.Atoi(maxIdleConnsStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DB_MAX_IDLE_CONNS %q: %w", maxIdleConnsStr, err)
	}

	connMaxLifetimeStr := strings.TrimSpace(os.Getenv("DB_CONN_MAX_LIFETIME"))
	if connMaxLifetimeStr == "" {
		connMaxLifetimeStr = "0s"
	}
	connMaxLifetime, err := time.ParseDuration(connMaxLifetimeStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid DB_CONN_MAX_LIFETIME %q: %w", strings.TrimSpace(os.Getenv("DB_CONN_MAX_LIFETIME")), err)
	}

	return Config{
		AppEnv:          appEnv,
		LogLevel:        level,
		HTTPAddr:        httpAddr,
		StaticDir:       staticDir,
		Driver:          driver,
		DSN:             dsn,
		Path:            path,
		MaxOpenConns:    maxOpenConns,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
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
