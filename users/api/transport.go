// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/absmach/supermq"
	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/oauth2"
	"github.com/absmach/supermq/users"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

// MakeHandler returns a HTTP handler for Users and Groups API endpoints.
func MakeHandler(svc users.Service, authn smqauthn.AuthNMiddleware, tokensvc grpcTokenV1.TokenServiceClient, selfRegister bool, mux *chi.Mux, logger *slog.Logger, instanceID string, pr *regexp.Regexp, idp supermq.IDProvider, cacheClient *redis.Client, providers ...oauth2.Provider) http.Handler {
	ctx := context.Background()
	deviceStore := NewRedisDeviceCodeStore(ctx, cacheClient)

	mux = usersHandler(svc, authn, tokensvc, selfRegister, mux, logger, pr, idp)
	mux = oauthHandler(mux, svc, tokensvc, deviceStore, providers...)
	mux = oauthDeviceHandler(mux, deviceStore, svc, tokensvc, providers...)

	mux.Get("/health", supermq.Health("users", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
