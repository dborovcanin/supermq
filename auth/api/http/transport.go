// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package http

import (
	"net/http"

	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/auth/api/http/groups"
	"github.com/mainflux/mainflux/auth/api/http/keys"
	"github.com/mainflux/mainflux/auth/api/http/policies"
	"github.com/mainflux/mainflux/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, logger logger.Logger) http.Handler {
	mux := bone.New()
	mux = keys.MakeHandler(svc, mux, logger)
	mux = groups.MakeHandler(svc, mux, logger)
	mux = policies.MakeHandler(svc, mux, logger)
	mux.GetFunc("/health", mainflux.Health("auth", ""))
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}
