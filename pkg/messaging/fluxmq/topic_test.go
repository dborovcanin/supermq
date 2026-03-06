// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import "testing"

func TestToMQTTTopic(t *testing.T) {
	cases := []struct {
		name     string
		topic    string
		expected string
	}{
		{
			name:     "dot style without prefix",
			topic:    "domain123.c.channel456.subtopic",
			expected: "m/domain123/c/channel456/subtopic",
		},
		{
			name:     "already mqtt style with prefix",
			topic:    "m/domain123/c/channel456/subtopic",
			expected: "m/domain123/c/channel456/subtopic",
		},
		{
			name:     "leading slash mqtt style",
			topic:    "/m/domain123/c/channel456/subtopic",
			expected: "m/domain123/c/channel456/subtopic",
		},
		{
			name:     "dot style with explicit m prefix",
			topic:    "m.domain123.c.channel456",
			expected: "m/domain123/c/channel456",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := toMQTTTopic("m", tc.topic)
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestToMQTTSubscribeTopic(t *testing.T) {
	cases := []struct {
		name     string
		topic    string
		expected string
	}{
		{
			name:     "all messages in mqtt format",
			topic:    "m/#",
			expected: "m/#",
		},
		{
			name:     "all messages in nats format",
			topic:    "m.>",
			expected: "m/#",
		},
		{
			name:     "nats wildcard without prefix",
			topic:    ">",
			expected: "m/#",
		},
		{
			name:     "dot style topic wildcard",
			topic:    "domain123.c.*",
			expected: "m/domain123/c/+",
		},
		{
			name:     "mqtt style with leading slash",
			topic:    "/m/domain123/c/#",
			expected: "m/domain123/c/#",
		},
		{
			name:     "empty topic defaults to all",
			topic:    "",
			expected: "m/#",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := toMQTTSubscribeTopic("m", tc.topic)
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
