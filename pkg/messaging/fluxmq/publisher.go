// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/absmach/fluxmq/client"
	"github.com/absmach/fluxmq/client/mqtt"
	"github.com/absmach/supermq/pkg/messaging"
	"google.golang.org/protobuf/proto"
)

var (
	ErrEmptyTopic = errors.New("empty topic")
	ErrConnect    = errors.New("failed to connect to FluxMQ broker")
)

var _ messaging.Publisher = (*publisher)(nil)

type publisher struct {
	client *client.Client
	options
}

// NewPublisher returns a FluxMQ message publisher.
func NewPublisher(ctx context.Context, url string, opts ...messaging.Option) (messaging.Publisher, error) {
	pub := &publisher{
		options: defaultOptions(),
	}

	for _, opt := range opts {
		if err := opt(pub); err != nil {
			return nil, err
		}
	}

	mqttOpts := mqtt.NewOptions()
	mqttOpts.Servers = []string{url}
	mqttOpts.CleanSession = true
	if pub.clientID != "" {
		mqttOpts.ClientID = pub.clientID
	} else {
		mqttOpts.ClientID = fmt.Sprintf("smq-publisher-%d", time.Now().UnixNano())
	}
	if pub.username != "" {
		mqttOpts.Username = pub.username
		mqttOpts.Password = pub.password
	}

	c, err := client.NewMQTT(mqttOpts)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnect, err)
	}

	if err := c.Connect(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrConnect, err)
	}

	pub.client = c
	return pub, nil
}

func (pub *publisher) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	mqttTopic := toMQTTTopic(pub.prefix, topic)
	qos := pub.qos
	return pub.client.Publish(ctx, mqttTopic, data, client.WithQoS(qos))
}

func (pub *publisher) Close() error {
	if pub.client != nil {
		return pub.client.Close(context.Background())
	}
	return nil
}

// toMQTTTopic converts a dot-separated NATS-style topic to slash-separated MQTT format.
// e.g., "domain123.c.channel456.subtopic" -> "m/domain123/c/channel456/subtopic"
func toMQTTTopic(prefix, topic string) string {
	topic = strings.TrimSpace(topic)
	topic = strings.TrimPrefix(topic, "/")
	topic = strings.ReplaceAll(topic, ".", "/")

	if strings.HasPrefix(topic, prefix+"/") {
		return topic
	}

	return prefix + "/" + topic
}

// fromMQTTTopic converts a slash-separated MQTT topic to a dot-separated subject,
// stripping the prefix.
// e.g., "m/domain123/c/channel456/subtopic" -> "domain123.c.channel456.subtopic"
func fromMQTTTopic(prefix, topic string) string {
	trimmed := strings.TrimPrefix(topic, prefix+"/")
	return strings.ReplaceAll(trimmed, "/", ".")
}
