// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux/internal/api"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/groups"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc groups.Service, mux *bone.Mux, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	mux.Post("/groups", otelhttp.NewHandler(kithttp.NewServer(
		CreateGroupEndpoint(svc),
		DecodeGroupCreate,
		api.EncodeResponse,
		opts...,
	), "create_group"))

	mux.Get("/groups/:groupID", otelhttp.NewHandler(kithttp.NewServer(
		ViewGroupEndpoint(svc),
		DecodeGroupRequest,
		api.EncodeResponse,
		opts...,
	), "view_group"))

	mux.Put("/groups/:groupID", otelhttp.NewHandler(kithttp.NewServer(
		UpdateGroupEndpoint(svc),
		DecodeGroupUpdate,
		api.EncodeResponse,
		opts...,
	), "update_group"))

	mux.Get("/users/:userID/memberships", otelhttp.NewHandler(kithttp.NewServer(
		ListMembershipsEndpoint(svc),
		DecodeListMembershipRequest,
		api.EncodeResponse,
		opts...,
	), "list_memberships"))

	mux.Get("/groups", otelhttp.NewHandler(kithttp.NewServer(
		ListGroupsEndpoint(svc),
		DecodeListGroupsRequest,
		api.EncodeResponse,
		opts...,
	), "list_groups"))

	mux.Get("/groups/:groupID/children", otelhttp.NewHandler(kithttp.NewServer(
		ListGroupsEndpoint(svc),
		DecodeListChildrenRequest,
		api.EncodeResponse,
		opts...,
	), "list_children"))

	mux.Get("/groups/:groupID/parents", otelhttp.NewHandler(kithttp.NewServer(
		ListGroupsEndpoint(svc),
		DecodeListParentsRequest,
		api.EncodeResponse,
		opts...,
	), "list_parents"))

	mux.Post("/groups/:groupID/enable", otelhttp.NewHandler(kithttp.NewServer(
		EnableGroupEndpoint(svc),
		DecodeChangeGroupStatus,
		api.EncodeResponse,
		opts...,
	), "enable_group"))

	mux.Post("/groups/:groupID/disable", otelhttp.NewHandler(kithttp.NewServer(
		DisableGroupEndpoint(svc),
		DecodeChangeGroupStatus,
		api.EncodeResponse,
		opts...,
	), "disable_group"))

	return mux
}
