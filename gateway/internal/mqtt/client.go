package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"cloudpico-gateway/internal/config"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client wraps an MQTT client with reconnection and backoff logic
type Client struct {
	client     mqtt.Client
	cfg        config.Config
	logger     *slog.Logger
	mu         sync.RWMutex
	connected  bool
	monitoring bool
	stopCh     chan struct{}
	wg         sync.WaitGroup
	stopOnce   sync.Once
}

// Telemetry represents station telemetry data
type Telemetry struct {
	StationID   string    `json:"station_id"`
	Timestamp   time.Time `json:"timestamp"`
	Temperature *float64  `json:"temperature_c,omitempty"`
	Humidity    *float64  `json:"humidity_pct,omitempty"`
	Pressure    *float64  `json:"pressure_hpa,omitempty"`
	Battery     *float64  `json:"battery_v,omitempty"`
	Sequence    *int      `json:"sequence,omitempty"`
}

// StationHealth represents station health/last-seen state
type StationHealth struct {
	StationID string    `json:"station_id"`
	LastSeen  time.Time `json:"last_seen"`
	Healthy   bool      `json:"healthy"`
}

// NewClient creates a new MQTT client with automatic reconnection
func NewClient(cfg config.Config, logger *slog.Logger) (*Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.MQTTBroker, cfg.MQTTPort))
	opts.SetClientID(cfg.MQTTClientID)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(60 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logger.Warn("mqtt connection lost", "error", err)
	})
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.Info("mqtt connected", "broker", cfg.MQTTBroker, "port", cfg.MQTTPort)
	})

	client := mqtt.NewClient(opts)

	c := &Client{
		client:    client,
		cfg:       cfg,
		logger:    logger,
		connected: false,
		stopCh:    make(chan struct{}),
	}

	return c, nil
}

// Connect establishes connection to the MQTT broker with exponential backoff
func (c *Client) Connect(ctx context.Context) error {
	// Check if already connected (quick check with lock)
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	backoff := time.Second
	maxBackoff := 60 * time.Second
	attempt := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return fmt.Errorf("client stopped")
		default:
		}

		// Attempt connection (without holding lock)
		token := c.client.Connect()
		if token.Wait() && token.Error() == nil {
			// Connection successful - update state with lock
			c.mu.Lock()
			// Double-check in case another goroutine connected while we were waiting
			if c.connected {
				c.mu.Unlock()
				return nil
			}
			c.connected = true

			// Start monitoring connection if not already monitoring
			if !c.monitoring {
				c.monitoring = true
				c.wg.Add(1)
				go c.monitorConnection(ctx)
			}
			c.mu.Unlock()

			c.logger.Info("mqtt connection established",
				"broker", c.cfg.MQTTBroker,
				"port", c.cfg.MQTTPort,
				"client_id", c.cfg.MQTTClientID,
			)

			return nil
		}

		attempt++
		c.logger.Warn("mqtt connection failed, retrying",
			"error", token.Error(),
			"attempt", attempt,
			"backoff", backoff,
		)

		// Release lock during backoff sleep to allow other operations
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return fmt.Errorf("client stopped")
		case <-time.After(backoff):
		}

		// Exponential backoff with jitter
		backoff = time.Duration(float64(backoff) * 1.5)
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		// Add jitter (±20%)
		jitter := time.Duration(float64(backoff) * 0.2 * (rand.Float64()*2 - 1))
		backoff = backoff + jitter
	}
}

// monitorConnection monitors the connection and updates internal state
func (c *Client) monitorConnection(ctx context.Context) {
	defer func() {
		c.mu.Lock()
		c.monitoring = false
		c.mu.Unlock()
		c.wg.Done()
	}()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			isConnected := c.client.IsConnected()
			c.mu.Lock()
			wasConnected := c.connected
			c.connected = isConnected
			c.mu.Unlock()

			if wasConnected && !isConnected {
				c.logger.Warn("mqtt connection lost, client will auto-reconnect")
			} else if !wasConnected && isConnected {
				c.logger.Info("mqtt connection restored")
			}
		}
	}
}

// PublishTelemetry publishes telemetry data to the station topic
func (c *Client) PublishTelemetry(stationID string, telemetry Telemetry) error {
	topic := fmt.Sprintf("stations/%s/telemetry", stationID)

	c.mu.RLock()
	connected := c.connected && c.client.IsConnected()
	c.mu.RUnlock()

	if !connected {
		return fmt.Errorf("mqtt client not connected")
	}

	telemetry.StationID = stationID
	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}

	data, err := json.Marshal(telemetry)
	if err != nil {
		return fmt.Errorf("marshal telemetry: %w", err)
	}

	token := c.client.Publish(topic, 1, false, data)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish timeout for topic %s", topic)
	}
	if token.Error() != nil {
		c.logger.Error("failed to publish telemetry",
			"topic", topic,
			"error", token.Error(),
		)
		return fmt.Errorf("publish telemetry: %w", token.Error())
	}

	c.logger.Debug("published telemetry",
		"topic", topic,
		"station_id", stationID,
	)

	return nil
}

// PublishStationHealth publishes station health/last-seen state
func (c *Client) PublishStationHealth(health StationHealth) error {
	topic := fmt.Sprintf("stations/%s/health", health.StationID)

	c.mu.RLock()
	connected := c.connected && c.client.IsConnected()
	c.mu.RUnlock()

	if !connected {
		return fmt.Errorf("mqtt client not connected")
	}

	if health.LastSeen.IsZero() {
		health.LastSeen = time.Now()
	}

	data, err := json.Marshal(health)
	if err != nil {
		return fmt.Errorf("marshal health: %w", err)
	}

	token := c.client.Publish(topic, 1, true, data) // Retained message
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish timeout for topic %s", topic)
	}
	if token.Error() != nil {
		c.logger.Error("failed to publish station health",
			"topic", topic,
			"error", token.Error(),
		)
		return fmt.Errorf("publish health: %w", token.Error())
	}

	c.logger.Debug("published station health",
		"topic", topic,
		"station_id", health.StationID,
		"last_seen", health.LastSeen,
		"healthy", health.Healthy,
	)

	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.client.IsConnected()
}

// Disconnect closes the MQTT connection
func (c *Client) Disconnect() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
	c.wg.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		c.client.Disconnect(250)
		c.connected = false
		c.logger.Info("mqtt disconnected")
	}
	c.monitoring = false
}
