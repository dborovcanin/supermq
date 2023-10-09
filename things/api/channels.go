// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/mainflux/mainflux/internal/api"
	"github.com/mainflux/mainflux/internal/apiutil"
	gapi "github.com/mainflux/mainflux/internal/groups/api"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/things"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func groupsHandler(svc groups.Service, tscv things.Service, r *chi.Mux, logger logger.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	r.Route("/channels", func(r chi.Router) {
		r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
			gapi.CreateGroupEndpoint(svc),
			gapi.DecodeGroupCreate,
			api.EncodeResponse,
			opts...,
		), "create_channel").ServeHTTP)

		r.Get("/{groupID}", otelhttp.NewHandler(kithttp.NewServer(
			gapi.ViewGroupEndpoint(svc),
			gapi.DecodeGroupRequest,
			api.EncodeResponse,
			opts...,
		), "view_channel").ServeHTTP)

		r.Put("/{groupID}", otelhttp.NewHandler(kithttp.NewServer(
			gapi.UpdateGroupEndpoint(svc),
			gapi.DecodeGroupUpdate,
			api.EncodeResponse,
			opts...,
		), "update_channel").ServeHTTP)

		r.Get("/{groupID}/things", otelhttp.NewHandler(kithttp.NewServer(
			listMembersEndpoint(tscv),
			decodeListMembersRequest,
			api.EncodeResponse,
			opts...,
		), "list_things_by_channel").ServeHTTP)

		r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
			gapi.ListGroupsEndpoint(svc, "users"),
			gapi.DecodeListGroupsRequest,
			api.EncodeResponse,
			opts...,
		), "list_channels").ServeHTTP)

		r.Post("/{groupID}/enable", otelhttp.NewHandler(kithttp.NewServer(
			gapi.EnableGroupEndpoint(svc),
			gapi.DecodeChangeGroupStatus,
			api.EncodeResponse,
			opts...,
		), "enable_channel").ServeHTTP)

		r.Post("/{groupID}/disable", otelhttp.NewHandler(kithttp.NewServer(
			gapi.DisableGroupEndpoint(svc),
			gapi.DecodeChangeGroupStatus,
			api.EncodeResponse,
			opts...,
		), "disable_channel").ServeHTTP)

		r.Post("/{groupID}/members", otelhttp.NewHandler(kithttp.NewServer(
			assignUsersGroupsEndpoint(svc),
			decodeAssignUsersGroupsRequest,
			api.EncodeResponse,
			opts...,
		), "assign_members").ServeHTTP)

		r.Delete("/{groupID}/members", otelhttp.NewHandler(kithttp.NewServer(
			unassignUsersGroupsEndpoint(svc),
			decodeUnassignUsersGroupsRequest,
			api.EncodeResponse,
			opts...,
		), "unassign_members").ServeHTTP)

		r.Post("/{groupID}/things/{thingID}", otelhttp.NewHandler(kithttp.NewServer(
			connectChannelThingEndpoint(svc),
			decodeConnectChannelThingRequest,
			api.EncodeResponse,
			opts...,
		), "connect_channel_thing").ServeHTTP)

		r.Delete("/{groupID}/things/{thingID}", otelhttp.NewHandler(kithttp.NewServer(
			disconnectChannelThingEndpoint(svc),
			decodeDisconnectChannelThingRequest,
			api.EncodeResponse,
			opts...,
		), "disconnect_channel_thing").ServeHTTP)
	})

	r.Get("/things/{memberID}/channels", otelhttp.NewHandler(kithttp.NewServer(
		gapi.ListGroupsEndpoint(svc, "things"),
		gapi.DecodeListGroupsRequest,
		api.EncodeResponse,
		opts...,
	), "list_channel_by_things").ServeHTTP)

	r.Post("/connect", otelhttp.NewHandler(kithttp.NewServer(
		connectEndpoint(svc),
		decodeConnectRequest,
		api.EncodeResponse,
		opts...,
	), "connect").ServeHTTP)

	r.Post("/disconnect", otelhttp.NewHandler(kithttp.NewServer(
		disconnectEndpoint(svc),
		decodeDisconnectRequest,
		api.EncodeResponse,
		opts...,
	), "disconnect").ServeHTTP)

	return r
}

func decodeAssignUsersGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := assignUsersGroupsRequest{
		token:   apiutil.ExtractBearerToken(r),
		groupID: chi.URLParam(r, "groupID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	return req, nil
}

func decodeUnassignUsersGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := unassignUsersGroupsRequest{
		token:   apiutil.ExtractBearerToken(r),
		groupID: chi.URLParam(r, "groupID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	return req, nil
}

func decodeConnectChannelThingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := connectChannelThingRequest{
		token:     apiutil.ExtractBearerToken(r),
		ThingID:   chi.URLParam(r, "thingID"),
		ChannelID: chi.URLParam(r, "groupID"),
	}
	return req, nil
}

func decodeDisconnectChannelThingRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := disconnectChannelThingRequest{
		token:     apiutil.ExtractBearerToken(r),
		ThingID:   chi.URLParam(r, "thingID"),
		ChannelID: chi.URLParam(r, "groupID"),
	}
	return req, nil
}

func decodeConnectRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := connectChannelThingRequest{
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	return req, nil
}

func decodeDisconnectRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := disconnectChannelThingRequest{
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	return req, nil
}
