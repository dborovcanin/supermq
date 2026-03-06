// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/absmach/fluxmq/client"
	"github.com/absmach/fluxmq/client/mqtt"
	"github.com/absmach/supermq/pkg/messaging"
	"google.golang.org/protobuf/proto"
)

var (
	ErrNotSubscribed = errors.New("not subscribed")
	ErrEmptyID       = errors.New("empty id")
)

var _ messaging.PubSub = (*pubsub)(nil)

type subscription struct {
	topic  string
	cancel context.CancelFunc
}

type pubsub struct {
	publisher
	logger *slog.Logger

	mu   sync.Mutex
	subs map[string]subscription // key: subscriberID+topic
}

// NewPubSub returns a FluxMQ-backed message publisher/subscriber.
func NewPubSub(ctx context.Context, url string, logger *slog.Logger, opts ...messaging.Option) (messaging.PubSub, error) {
	ps := &pubsub{
		publisher: publisher{
			options: defaultOptions(),
		},
		logger: logger,
		subs:   make(map[string]subscription),
	}

	for _, opt := range opts {
		if err := opt(ps); err != nil {
			return nil, err
		}
	}

	mqttOpts := mqtt.NewOptions()
	mqttOpts.Servers = []string{url}
	mqttOpts.CleanSession = true
	if ps.clientID != "" {
		mqttOpts.ClientID = ps.clientID
	} else {
		mqttOpts.ClientID = fmt.Sprintf("smq-pubsub-%d", time.Now().UnixNano())
	}
	if ps.username != "" {
		mqttOpts.Username = ps.username
		mqttOpts.Password = ps.password
	}

	c, err := client.NewMQTT(mqttOpts)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnect, err)
	}

	if err := c.Connect(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnect, err)
	}

	ps.client = c
	return ps, nil
}

func (ps *pubsub) Subscribe(ctx context.Context, cfg messaging.SubscriberConfig) error {
	if cfg.ID == "" {
		return ErrEmptyID
	}
	if cfg.Topic == "" {
		return ErrEmptyTopic
	}

	mqttTopic := toMQTTSubscribeTopic(ps.prefix, cfg.Topic)
	subKey := subKey(cfg.ID, cfg.Topic)

	handler := ps.messageHandler(cfg.Handler)

	qos := ps.qos
	if err := ps.client.Subscribe(ctx, mqttTopic, handler, client.WithQoS(qos)); err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", mqttTopic, err)
	}

	ps.mu.Lock()
	ps.subs[subKey] = subscription{topic: mqttTopic}
	ps.mu.Unlock()

	return nil
}

func (ps *pubsub) Unsubscribe(ctx context.Context, id, topic string) error {
	if id == "" {
		return ErrEmptyID
	}
	if topic == "" {
		return ErrEmptyTopic
	}

	key := subKey(id, topic)

	ps.mu.Lock()
	sub, ok := ps.subs[key]
	if !ok {
		ps.mu.Unlock()
		return ErrNotSubscribed
	}
	delete(ps.subs, key)
	ps.mu.Unlock()

	if sub.cancel != nil {
		sub.cancel()
	}

	return ps.client.Unsubscribe(ctx, sub.topic)
}

func (ps *pubsub) Close() error {
	ps.mu.Lock()
	for _, sub := range ps.subs {
		if sub.cancel != nil {
			sub.cancel()
		}
	}
	ps.subs = make(map[string]subscription)
	ps.mu.Unlock()

	if ps.client != nil {
		return ps.client.Close(context.Background())
	}
	return nil
}

func (ps *pubsub) messageHandler(h messaging.MessageHandler) client.MessageHandler {
	return func(m *client.Message) {
		if m == nil {
			return
		}

		var msg messaging.Message
		if err := proto.Unmarshal(m.Payload, &msg); err != nil {
			ps.logger.Warn("failed to unmarshal message",
				slog.String("topic", m.Topic),
				slog.String("error", err.Error()))
			return
		}

		if err := h.Handle(&msg); err != nil {
			ps.logger.Warn("failed to handle message",
				slog.String("topic", m.Topic),
				slog.String("error", err.Error()))
		}
	}
}

func subKey(id, topic string) string {
	return id + ":" + topic
}

// toMQTTSubscribeTopic converts a NATS-style wildcard subscribe topic to MQTT format.
// e.g., "m.>" -> "m/#", "m.domain.c.*" -> "m/domain/c/+"
func toMQTTSubscribeTopic(prefix, topic string) string {
	mqttTopic := prefix + "/" + strings.ReplaceAll(topic, ".", "/")
	mqttTopic = strings.ReplaceAll(mqttTopic, ">", "#")
	mqttTopic = strings.ReplaceAll(mqttTopic, "*", "+")
	return mqttTopic
}
