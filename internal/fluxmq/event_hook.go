// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/absmach/fluxmq/broker"
	"github.com/absmach/supermq/pkg/messaging"
)

var _ broker.EventHook = (*EventHook)(nil)

// EventHook publishes FluxMQ lifecycle events to the SuperMQ event store (NATS).
type EventHook struct {
	publisher messaging.Publisher
	prefix    string
	logger    *slog.Logger
}

// NewEventHook creates an event hook that publishes to the given messaging publisher.
// The prefix controls the event topic namespace (e.g., "events.supermq.mqtt").
func NewEventHook(publisher messaging.Publisher, prefix string, logger *slog.Logger) *EventHook {
	if logger == nil {
		logger = slog.Default()
	}
	return &EventHook{
		publisher: publisher,
		prefix:    prefix,
		logger:    logger,
	}
}

func (h *EventHook) OnConnect(ctx context.Context, clientID, username, protocol string) error {
	return h.publish(ctx, "client_connect", map[string]interface{}{
		"client_id": clientID,
		"username":  username,
		"protocol":  protocol,
	})
}

func (h *EventHook) OnDisconnect(ctx context.Context, clientID, reason string) error {
	return h.publish(ctx, "client_disconnect", map[string]interface{}{
		"client_id": clientID,
		"reason":    reason,
	})
}

func (h *EventHook) OnSubscribe(ctx context.Context, clientID, topic string, qos byte) error {
	return h.publish(ctx, "client_subscribe", map[string]interface{}{
		"client_id": clientID,
		"topic":     topic,
		"qos":       qos,
	})
}

func (h *EventHook) OnUnsubscribe(ctx context.Context, clientID, topic string) error {
	return h.publish(ctx, "client_unsubscribe", map[string]interface{}{
		"client_id": clientID,
		"topic":     topic,
	})
}

func (h *EventHook) OnPublish(ctx context.Context, clientID, topic string, qos byte, payload []byte) error {
	return h.publish(ctx, "publish", map[string]interface{}{
		"client_id":    clientID,
		"topic":        topic,
		"qos":          qos,
		"payload_size": len(payload),
	})
}

func (h *EventHook) Close() error {
	return nil
}

func (h *EventHook) publish(ctx context.Context, operation string, data map[string]interface{}) error {
	data["operation"] = operation
	data["occurred_at"] = time.Now().UnixNano()

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	topic := h.prefix + "." + operation

	msg := &messaging.Message{
		Payload: payload,
		Created: time.Now().UnixNano(),
	}

	if err := h.publisher.Publish(ctx, topic, msg); err != nil {
		h.logger.Warn("failed to publish event",
			slog.String("operation", operation),
			slog.String("error", err.Error()))
		return err
	}

	return nil
}
