package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"cloudpico-server/internal/config"
	cloudpico_shared "cloudpico-shared/types"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Subscriber struct {
	client    mqtt.Client
	cfg       config.Config
	logger    *slog.Logger
	mu        sync.RWMutex
	connected bool

	stopCh   chan struct{}
	stopOnce sync.Once

	// MessageHandler is called for each valid telemetry message
	MessageHandler func(telemetry cloudpico_shared.Telemetry) error
}

// MQTTSubscriber interface for attaching message handlers
type MQTTSubscriber interface {
	SetMessageHandler(handler func(telemetry cloudpico_shared.Telemetry) error)
}

// SetMessageHandler sets the message handler for telemetry messages
func (s *Subscriber) SetMessageHandler(handler func(telemetry cloudpico_shared.Telemetry) error) {
	s.MessageHandler = handler
}

func NewSubscriber(cfg config.Config, logger *slog.Logger) (*Subscriber, error) {
	s := &Subscriber{
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
		s.setConnected(true)
		logger.Info("mqtt connected", "broker", cfg.MQTTBroker, "port", cfg.MQTTPort)
	})

	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		s.setConnected(false)
		logger.Warn("mqtt connection lost", "error", err)
	})

	s.client = mqtt.NewClient(opts)
	return s, nil
}

// Connect establishes connection to the MQTT broker and subscribes to the configured topic.
func (s *Subscriber) Connect(ctx context.Context) error {
	// Fail fast if already stopped.
	select {
	case <-s.stopCh:
		return fmt.Errorf("subscriber stopped")
	default:
	}

	// Fast path.
	if s.IsConnected() {
		return nil
	}

	// Start connect attempt.
	token := s.client.Connect()

	// Wait in a ctx/stop-aware loop.
	const poll = 200 * time.Millisecond
	for {
		if token.WaitTimeout(poll) {
			if err := token.Error(); err != nil {
				return fmt.Errorf("mqtt connect: %w", err)
			}
			// OnConnectHandler sets connected=true.
			break
		}

		select {
		case <-ctx.Done():
			s.client.Disconnect(0)
			return ctx.Err()
		case <-s.stopCh:
			s.client.Disconnect(0)
			return fmt.Errorf("subscriber stopped")
		default:
		}
	}

	// Subscribe to the topic
	if err := s.subscribe(); err != nil {
		s.client.Disconnect(0)
		return fmt.Errorf("subscribe: %w", err)
	}

	return nil
}

func (s *Subscriber) subscribe() error {
	if !s.IsConnected() {
		return fmt.Errorf("mqtt client not connected")
	}

	topic := s.cfg.MQTTTopic
	qos := byte(1) // At least once delivery

	// Set up message handler
	messageHandler := func(client mqtt.Client, msg mqtt.Message) {
		s.handleMessage(msg.Topic(), msg.Payload())
	}

	token := s.client.Subscribe(topic, qos, messageHandler)
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("subscribe timeout for topic %s", topic)
	}
	if token.Error() != nil {
		return fmt.Errorf("subscribe to %s: %w", topic, token.Error())
	}

	s.logger.Info("subscribed to mqtt topic", "topic", topic, "qos", qos)
	return nil
}

func (s *Subscriber) handleMessage(topic string, payload []byte) {
	s.logger.Debug("received mqtt message", "topic", topic, "size", len(payload))

	// Parse telemetry message
	var telemetry cloudpico_shared.Telemetry
	if err := json.Unmarshal(payload, &telemetry); err != nil {
		s.logger.Warn("failed to parse telemetry message",
			"topic", topic,
			"error", err,
			"payload", string(payload),
		)
		return
	}

	// Validate telemetry
	if err := s.validateTelemetry(telemetry); err != nil {
		s.logger.Warn("invalid telemetry message",
			"topic", topic,
			"station_id", telemetry.StationID,
			"error", err,
		)
		return
	}

	// Call the message handler if set
	if s.MessageHandler != nil {
		if err := s.MessageHandler(telemetry); err != nil {
			s.logger.Error("message handler failed",
				"topic", topic,
				"station_id", telemetry.StationID,
				"error", err,
			)
		} else {
			s.logger.Debug("processed telemetry message",
				"station_id", telemetry.StationID,
				"timestamp", telemetry.Timestamp,
			)
		}
	}
}

func (s *Subscriber) validateTelemetry(t cloudpico_shared.Telemetry) error {
	// Validate required fields
	if t.StationID == "" {
		return fmt.Errorf("station_id is required")
	}

	if t.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	// Validate optional fields if present
	if t.Humidity != nil {
		if *t.Humidity < 0 || *t.Humidity > 100 {
			return fmt.Errorf("humidity_pct out of range: %f (must be 0-100)", *t.Humidity)
		}
	}

	if t.Pressure != nil {
		if *t.Pressure <= 0 {
			return fmt.Errorf("pressure_hpa must be positive: %f", *t.Pressure)
		}
	}

	// At least one sensor reading should be present
	if t.Temperature == nil && t.Humidity == nil && t.Pressure == nil {
		return fmt.Errorf("at least one sensor reading (temperature, humidity, or pressure) is required")
	}

	return nil
}

// IsConnected returns whether the client is connected.
func (s *Subscriber) IsConnected() bool {
	s.mu.RLock()
	connected := s.connected
	s.mu.RUnlock()
	return connected && s.client.IsConnected()
}

// Disconnect stops the subscriber and closes the MQTT connection.
// Idempotent and safe to call multiple times.
func (s *Subscriber) Disconnect() {
	// Signal shutdown once (unblocks any Connect loops).
	s.stopOnce.Do(func() { close(s.stopCh) })

	// Unsubscribe before disconnecting
	if s.client != nil && s.IsConnected() {
		token := s.client.Unsubscribe(s.cfg.MQTTTopic)
		token.WaitTimeout(2 * time.Second)
	}

	// Disconnect without holding s.mu to avoid lock contention/deadlocks.
	if s.client != nil {
		s.client.Disconnect(250)
	}

	// Update our internal state.
	s.setConnected(false)
	s.logger.Info("mqtt subscriber disconnected")
}

func (s *Subscriber) setConnected(v bool) {
	s.mu.Lock()
	s.connected = v
	s.mu.Unlock()
}
