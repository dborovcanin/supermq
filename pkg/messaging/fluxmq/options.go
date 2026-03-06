// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"errors"

	"github.com/absmach/supermq/pkg/messaging"
)

var ErrInvalidType = errors.New("invalid type")

const defaultPrefix = "m"

type options struct {
	prefix   string
	clientID string
	username string
	password string
	qos      byte
}

func defaultOptions() options {
	return options{
		prefix: defaultPrefix,
		qos:    1,
	}
}

// Prefix sets the topic prefix for the publisher or subscriber.
func Prefix(prefix string) messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			v.prefix = prefix
		case *pubsub:
			v.prefix = prefix
		default:
			return ErrInvalidType
		}
		return nil
	}
}

// ClientID sets the MQTT client ID.
func ClientID(id string) messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			v.clientID = id
		case *pubsub:
			v.clientID = id
		default:
			return ErrInvalidType
		}
		return nil
	}
}

// Credentials sets the MQTT username and password.
func Credentials(username, password string) messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			v.username = username
			v.password = password
		case *pubsub:
			v.username = username
			v.password = password
		default:
			return ErrInvalidType
		}
		return nil
	}
}

// QoS sets the default MQTT QoS level for publish operations.
func QoS(qos byte) messaging.Option {
	return func(val any) error {
		switch v := val.(type) {
		case *publisher:
			v.qos = qos
		case *pubsub:
			v.qos = qos
		default:
			return ErrInvalidType
		}
		return nil
	}
}
