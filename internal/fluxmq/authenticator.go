// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/absmach/fluxmq/broker"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
)

const (
	defaultAuthTimeout     = 2 * time.Second
	cbMaxRequests          = 3
	cbOpenInterval         = 10 * time.Second
	cbFailureThreshold     = 5
)

var _ broker.Authenticator = (*Authenticator)(nil)

// Authenticator validates client credentials against SuperMQ's clients gRPC service.
type Authenticator struct {
	client  grpcClientsV1.ClientsServiceClient
	timeout time.Duration
	cb      *gobreaker.CircuitBreaker
	logger  *slog.Logger
}

// NewAuthenticator creates a SuperMQ-backed authenticator for FluxMQ.
func NewAuthenticator(conn *grpc.ClientConn, timeout time.Duration, logger *slog.Logger) *Authenticator {
	if timeout <= 0 {
		timeout = defaultAuthTimeout
	}
	if logger == nil {
		logger = slog.Default()
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "smq-authenticator",
		MaxRequests: cbMaxRequests,
		Interval:    0,
		Timeout:     cbOpenInterval,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cbFailureThreshold
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Warn("smq authenticator circuit breaker state changed",
				slog.String("from", from.String()),
				slog.String("to", to.String()))
		},
	})

	return &Authenticator{
		client:  grpcClientsV1.NewClientsServiceClient(conn),
		timeout: timeout,
		cb:      cb,
		logger:  logger,
	}
}

// Authenticate validates client credentials via SuperMQ clients service.
// The clientID from MQTT CONNECT maps to the SuperMQ client ID.
// Username and password are passed as-is for BasicAuth validation.
func (a *Authenticator) Authenticate(clientID, username, secret string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	// Pack credentials the same way SuperMQ MQTT handler does:
	// Token = "Basic" + base64(username:secret)
	token := authn.AuthPack(authn.BasicAuth, username, secret)

	req := &grpcClientsV1.AuthnReq{
		Token: token,
	}

	result, err := a.cb.Execute(func() (any, error) {
		return a.client.Authenticate(ctx, req)
	})
	if err != nil {
		a.logger.Warn("smq_authenticate_rpc_failed",
			slog.String("client_id", clientID),
			slog.String("error", err.Error()))
		return false, nil
	}

	res := result.(*grpcClientsV1.AuthnRes)
	if !res.GetAuthenticated() {
		a.logger.Debug("smq_authenticate_denied",
			slog.String("client_id", clientID))
		return false, nil
	}

	return true, nil
}

func normalizeClientID(clientID string) string {
	prefixes := []string{
		broker.AMQP091ClientPrefix,
		broker.AMQP1ClientPrefix,
		broker.HTTPClientPrefix,
		broker.CoAPClientPrefix,
	}
	for _, p := range prefixes {
		if after, ok := strings.CutPrefix(clientID, p); ok {
			return after
		}
	}
	return clientID
}
