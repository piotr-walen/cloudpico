package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv       string
	LogLevel     slog.Level
	MQTTBroker   string
	MQTTPort     int
	MQTTClientID string

	BME280Address      uint16
	SensorPollInterval time.Duration
	DeviceStationID    string
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

	mqttBroker := strings.TrimSpace(os.Getenv("MQTT_BROKER"))
	if mqttBroker == "" {
		mqttBroker = "localhost"
	}

	mqttPortStr := strings.TrimSpace(os.Getenv("MQTT_PORT"))
	if mqttPortStr == "" {
		mqttPortStr = "1883"
	}
	mqttPort, err := strconv.Atoi(mqttPortStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid MQTT_PORT %q: %w", mqttPortStr, err)
	}

	mqttClientID := strings.TrimSpace(os.Getenv("MQTT_CLIENT_ID"))
	if mqttClientID == "" {
		mqttClientID = "cloudpico-gateway"
	}

	bme280AddressStr := strings.TrimSpace(os.Getenv("BME280_ADDRESS"))
	if bme280AddressStr == "" {
		bme280AddressStr = "0x76"
	}
	bme280Address, err := strconv.ParseUint(bme280AddressStr, 0, 16)
	if err != nil {
		return Config{}, fmt.Errorf("invalid BME280_ADDRESS %q: %w", bme280AddressStr, err)
	}

	sensorPollIntervalStr := strings.TrimSpace(os.Getenv("SENSOR_POLL_INTERVAL"))
	if sensorPollIntervalStr == "" {
		sensorPollIntervalStr = "1s"
	}
	sensorPollInterval, err := time.ParseDuration(sensorPollIntervalStr)
	if err != nil {
		return Config{}, fmt.Errorf("invalid SENSOR_POLL_INTERVAL %q: %w", sensorPollIntervalStr, err)
	}
	if sensorPollInterval <= 0 {
		return Config{}, fmt.Errorf("SENSOR_POLL_INTERVAL must be positive, got %v", sensorPollInterval)
	}

	deviceStationID := strings.TrimSpace(os.Getenv("DEVICE_STATION_ID"))
	if deviceStationID == "" {
		deviceStationID = "home"
	}

	return Config{
		AppEnv:             appEnv,
		LogLevel:           level,
		MQTTBroker:         mqttBroker,
		MQTTPort:           mqttPort,
		MQTTClientID:       mqttClientID,
		BME280Address:      uint16(bme280Address),
		SensorPollInterval: sensorPollInterval,
		DeviceStationID:    deviceStationID,
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
