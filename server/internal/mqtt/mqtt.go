package mqtt

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"cloudpico-server/internal/config"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MessageHandler func(mqtt.Message) error
type Subscriber struct {
	client    mqtt.Client
	cfg       config.Config
	mu        sync.RWMutex
	connected bool

	stopCh chan struct{}

	messageHandler func(mqtt.Message) error
}

func NewSubscriber(cfg config.Config) *Subscriber {
	return &Subscriber{
		cfg: cfg,
	}
}

func (s *Subscriber) setConnected(connected bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = connected
}

// Connected reports whether the subscriber is currently connected to the MQTT broker.
func (s *Subscriber) Connected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

func (s *Subscriber) connect(ctx context.Context) error {
	token := s.client.Connect()
	const poll = 200 * time.Millisecond
	for {
		if token.WaitTimeout(poll) {
			if err := token.Error(); err != nil {
				return fmt.Errorf("mqtt connect: %w", err)
			}
			if s.client.IsConnected() {
				s.setConnected(true)
			}
			slog.Info("mqtt connected", "broker", s.cfg.MQTTBroker, "port", s.cfg.MQTTPort)
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
	return nil
}

func (s *Subscriber) messageCallback(_ mqtt.Client, msg mqtt.Message) {
	if s == nil || msg == nil || s.messageHandler == nil {
		return
	}
	_ = s.messageHandler(msg)
}

func (s *Subscriber) Subscribe(ctx context.Context) error {
	token := s.client.Subscribe(s.cfg.MQTTTopic, 1, s.messageCallback)

	done := make(chan struct{})
	go func() {
		token.Wait()
		close(done)
	}()

	select {
	case <-done:
		if err := token.Error(); err != nil {
			return fmt.Errorf("mqtt subscribe: %w", err)
		}
		return nil
	case <-ctx.Done():
		s.client.Unsubscribe(s.cfg.MQTTTopic)
		return ctx.Err()
	}
}

func getOptions(s *Subscriber) *mqtt.ClientOptions {
	cfg := s.cfg
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.MQTTBroker, cfg.MQTTPort))
	opts.SetClientID(cfg.MQTTClientID)
	// Persistent session so the broker queues QoS 1 messages when we're disconnected
	// and delivers them when we reconnect. Requires a stable, unique ClientID.
	opts.SetCleanSession(false)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(60 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		s.setConnected(true)
		slog.Info("mqtt connected", "broker", cfg.MQTTBroker, "port", cfg.MQTTPort)
		// Subscribe immediately on connect. The broker may send queued messages right after
		// CONNACK, before we would otherwise call Subscribe() from run.go. If we don't
		// subscribe here (synchronously), those queued messages can be dropped. Must be
		// synchronous so SUBSCRIBE is sent before the handler returns.
		if s.messageHandler != nil {
			token := c.Subscribe(s.cfg.MQTTTopic, 1, s.messageCallback)
			token.Wait()
			if err := token.Error(); err != nil {
				slog.Error("mqtt subscribe on connect failed", "topic", s.cfg.MQTTTopic, "error", err)
			}
		}
	})
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		s.setConnected(false)
		slog.Warn("mqtt connection lost", "broker", cfg.MQTTBroker, "port", cfg.MQTTPort)
	})
	return opts
}

func (s *Subscriber) Connect(ctx context.Context) error {
	opts := getOptions(s)
	s.client = mqtt.NewClient(opts)

	if err := s.connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	return nil
}

func (s *Subscriber) SetMessageHandler(handler MessageHandler) {
	s.messageHandler = handler
}

func (s *Subscriber) Disconnect() {
	s.client.Disconnect(0)
}
