// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"fmt"
	"strings"

	"github.com/absmach/fluxmq/broker"
	"github.com/absmach/supermq/pkg/messaging"
)

var _ broker.TopicRewriter = (*TopicRewriter)(nil)

// TopicRewriter resolves human-readable domain/channel names to UUIDs
// in SuperMQ message topics (m/<domain>/c/<channel>/...).
type TopicRewriter struct {
	resolver messaging.TopicResolver
}

// NewTopicRewriter creates a new TopicRewriter using the given resolver.
func NewTopicRewriter(resolver messaging.TopicResolver) *TopicRewriter {
	return &TopicRewriter{resolver: resolver}
}

func (r *TopicRewriter) RewritePublish(ctx context.Context, clientID, topic string) (string, error) {
	return r.rewrite(ctx, topic)
}

func (r *TopicRewriter) RewriteSubscribe(ctx context.Context, clientID, topic string) (string, error) {
	return r.rewrite(ctx, topic)
}

func (r *TopicRewriter) rewrite(ctx context.Context, topic string) (string, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(topic), "/")
	if trimmed == "" {
		return topic, nil
	}

	parts := strings.Split(trimmed, "/")
	if parts[0] != string(messaging.MsgTopicPrefix) {
		return topic, nil
	}

	if len(parts) < 4 || parts[2] != string(messaging.ChannelTopicPrefix) {
		return topic, nil
	}

	domain := parts[1]
	channel := parts[3]

	// If both are already UUIDs, no resolution needed
	if isUUID(domain) && isUUID(channel) {
		return topic, nil
	}

	domainID, channelID, _, err := r.resolver.Resolve(ctx, domain, channel)
	if err != nil {
		return "", fmt.Errorf("failed to resolve topic %q: %w", topic, err)
	}

	// Reconstruct topic with resolved IDs
	parts[1] = domainID
	parts[3] = channelID

	return strings.Join(parts, "/"), nil
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
