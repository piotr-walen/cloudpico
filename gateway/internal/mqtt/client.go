package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"cloudpico-gateway/internal/config"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	client    mqtt.Client
	cfg       config.Config
	logger    *slog.Logger
	mu        sync.RWMutex
	connected bool

	stopCh   chan struct{}
	stopOnce sync.Once
}

type Telemetry struct {
	StationID   string    `json:"station_id"`
	Timestamp   time.Time `json:"timestamp"`
	Temperature *float64  `json:"temperature_c,omitempty"`
	Humidity    *float64  `json:"humidity_pct,omitempty"`
	Pressure    *float64  `json:"pressure_hpa,omitempty"`
	Battery     *float64  `json:"battery_v,omitempty"`
	Sequence    *int      `json:"sequence,omitempty"`
}

type StationHealth struct {
	StationID string    `json:"station_id"`
	LastSeen  time.Time `json:"last_seen"`
	Healthy   bool      `json:"healthy"`
}

func NewClient(cfg config.Config, logger *slog.Logger) (*Client, error) {
	c := &Client{
		cfg:    cfg,
		logger: logger,
		stopCh: make(chan struct{}),
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.MQTTBroker, cfg.MQTTPort))
	opts.SetClientID(cfg.MQTTClientID)

	// Session settings
	opts.SetCleanSession(true)

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(60 * time.Second)

	// Keepalive / timeouts
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	// Callbacks keep internal state accurate
	opts.SetOnConnectHandler(func(_ mqtt.Client) {
		c.setConnected(true)
		logger.Info("mqtt connected", "broker", cfg.MQTTBroker, "port", cfg.MQTTPort)
	})

	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		c.setConnected(false)
		logger.Warn("mqtt connection lost", "error", err)
	})

	c.client = mqtt.NewClient(opts)
	return c, nil
}

// Connect establishes connection to the MQTT broker.
// This function waits for the initial connection, and respects ctx and Disconnect().
func (c *Client) Connect(ctx context.Context) error {
	// Fail fast if already stopped.
	select {
	case <-c.stopCh:
		return fmt.Errorf("client stopped")
	default:
	}

	// Fast path.
	if c.IsConnected() {
		return nil
	}

	// Start connect attempt. With ConnectRetry(true), it may keep retrying internally.
	token := c.client.Connect()

	// Wait in a ctx/stop-aware loop.
	const poll = 200 * time.Millisecond
	for {
		if token.WaitTimeout(poll) {
			if err := token.Error(); err != nil {
				return fmt.Errorf("mqtt connect: %w", err)
			}
			// OnConnectHandler sets connected=true.
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stopCh:
			return fmt.Errorf("client stopped")
		default:
		}
	}
}

// PublishTelemetry publishes telemetry data to the station topic.
func (c *Client) PublishTelemetry(stationID string, telemetry Telemetry) error {
	if !c.IsConnected() {
		return fmt.Errorf("mqtt client not connected")
	}

	topic := fmt.Sprintf("stations/%s/telemetry", stationID)

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
		c.logger.Error("failed to publish telemetry", "topic", topic, "error", token.Error())
		return fmt.Errorf("publish telemetry: %w", token.Error())
	}

	c.logger.Debug("published telemetry", "topic", topic, "station_id", stationID)
	return nil
}

// PublishStationHealth publishes station health/last-seen state.
func (c *Client) PublishStationHealth(health StationHealth) error {
	if !c.IsConnected() {
		return fmt.Errorf("mqtt client not connected")
	}

	topic := fmt.Sprintf("stations/%s/health", health.StationID)

	if health.LastSeen.IsZero() {
		health.LastSeen = time.Now()
	}

	data, err := json.Marshal(health)
	if err != nil {
		return fmt.Errorf("marshal health: %w", err)
	}

	token := c.client.Publish(topic, 1, true, data) // retained
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish timeout for topic %s", topic)
	}
	if token.Error() != nil {
		c.logger.Error("failed to publish station health", "topic", topic, "error", token.Error())
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

// IsConnected returns whether the client is connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	connected := c.connected
	c.mu.RUnlock()
	return connected && c.client.IsConnected()
}

// Disconnect stops the client and closes the MQTT connection.
// Idempotent and safe to call multiple times.
// After Disconnect, Connect() will return "client stopped".
func (c *Client) Disconnect() {
	// Signal shutdown once (unblocks any Connect loops).
	c.stopOnce.Do(func() { close(c.stopCh) })

	// Disconnect without holding c.mu to avoid lock contention/deadlocks.
	// Paho Disconnect quiesces in-flight work for the given ms.
	if c.client != nil {
		// Even if already disconnected, this is safe.
		c.client.Disconnect(250)
	}

	// Update our internal state.
	c.setConnected(false)
	c.logger.Info("mqtt disconnected")
}

func (c *Client) setConnected(v bool) {
	c.mu.Lock()
	c.connected = v
	c.mu.Unlock()
}
