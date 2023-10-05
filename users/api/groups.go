// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/mainflux/mainflux/internal/api"
	"github.com/mainflux/mainflux/internal/apiutil"
	gapi "github.com/mainflux/mainflux/internal/groups/api"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/groups"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MakeHandler returns a HTTP handler for Groups API endpoints.
func groupsHandler(svc groups.Service, r *chi.Mux, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	r.Route("/groups", func(r chi.Router) {
		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			gapi.CreateGroupEndpoint(svc),
			gapi.DecodeGroupCreate,
			api.EncodeResponse,
			opts...,
		), "create_group").ServeHTTP)

		r.Get("/{groupID}", otelhttp.NewHandler(kithttp.NewServer(
			gapi.ViewGroupEndpoint(svc),
			gapi.DecodeGroupRequest,
			api.EncodeResponse,
			opts...,
		), "view_group").ServeHTTP)

		r.Put("/{groupID}", otelhttp.NewHandler(kithttp.NewServer(
			gapi.UpdateGroupEndpoint(svc),
			gapi.DecodeGroupUpdate,
			api.EncodeResponse,
			opts...,
		), "update_group").ServeHTTP)

		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			gapi.ListGroupsEndpoint(svc, "users"),
			gapi.DecodeListGroupsRequest,
			api.EncodeResponse,
			opts...,
		), "list_groups").ServeHTTP)

		r.Get("/{groupID}/children", otelhttp.NewHandler(kithttp.NewServer(
			gapi.ListGroupsEndpoint(svc, "users"),
			gapi.DecodeListChildrenRequest,
			api.EncodeResponse,
			opts...,
		), "list_children").ServeHTTP)

		r.Get("/{groupID}/parents", otelhttp.NewHandler(kithttp.NewServer(
			gapi.ListGroupsEndpoint(svc, "users"),
			gapi.DecodeListParentsRequest,
			api.EncodeResponse,
			opts...,
		), "list_parents").ServeHTTP)

		r.Post("/{groupID}/enable", otelhttp.NewHandler(kithttp.NewServer(
			gapi.EnableGroupEndpoint(svc),
			gapi.DecodeChangeGroupStatus,
			api.EncodeResponse,
			opts...,
		), "enable_group").ServeHTTP)

		r.Post("/{groupID}/disable", otelhttp.NewHandler(kithttp.NewServer(
			gapi.DisableGroupEndpoint(svc),
			gapi.DecodeChangeGroupStatus,
			api.EncodeResponse,
			opts...,
		), "disable_group").ServeHTTP)

		r.Post("/{groupID}/members", otelhttp.NewHandler(kithttp.NewServer(
			gapi.AssignMembersEndpoint(svc, "", "users"),
			gapi.DecodeAssignMembersRequest,
			api.EncodeResponse,
			opts...,
		), "assign_members").ServeHTTP)

		r.Delete("/{groupID}/members", otelhttp.NewHandler(kithttp.NewServer(
			gapi.UnassignMembersEndpoint(svc, "", "users"),
			gapi.DecodeUnassignMembersRequest,
			api.EncodeResponse,
			opts...,
		), "unassign_members").ServeHTTP)

		r.Get("/{groupID}/members", otelhttp.NewHandler(kithttp.NewServer(
			gapi.ListMembersEndpoint(svc, "users"),
			gapi.DecodeListMembersRequest,
			api.EncodeResponse,
			opts...,
		), "list_members").ServeHTTP)
	})

	return r
}
