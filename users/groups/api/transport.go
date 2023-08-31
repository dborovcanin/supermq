// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux/internal/api"
	"github.com/mainflux/mainflux/internal/apiutil"
	gapi "github.com/mainflux/mainflux/internal/groups/api"
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
		gapi.CreateGroupEndpoint(svc),
		// createGroupEndpoint(svc),
		gapi.DecodeGroupCreate,
		api.EncodeResponse,
		opts...,
	), "create_group"))

	mux.Get("/groups/:groupID", otelhttp.NewHandler(kithttp.NewServer(
		gapi.ViewGroupEndpoint(svc),
		gapi.DecodeGroupRequest,
		api.EncodeResponse,
		opts...,
	), "view_group"))

	mux.Put("/groups/:groupID", otelhttp.NewHandler(kithttp.NewServer(
		gapi.UpdateGroupEndpoint(svc),
		gapi.DecodeGroupUpdate,
		api.EncodeResponse,
		opts...,
	), "update_group"))

	mux.Get("/users/:userID/memberships", otelhttp.NewHandler(kithttp.NewServer(
		gapi.ListMembershipsEndpoint(svc),
		gapi.DecodeListMembershipRequest,
		api.EncodeResponse,
		opts...,
	), "list_memberships"))

	mux.Get("/groups", otelhttp.NewHandler(kithttp.NewServer(
		gapi.ListGroupsEndpoint(svc),
		gapi.DecodeListGroupsRequest,
		api.EncodeResponse,
		opts...,
	), "list_groups"))

	mux.Get("/groups/:groupID/children", otelhttp.NewHandler(kithttp.NewServer(
		gapi.ListGroupsEndpoint(svc),
		gapi.DecodeListChildrenRequest,
		api.EncodeResponse,
		opts...,
	), "list_children"))

	mux.Get("/groups/:groupID/parents", otelhttp.NewHandler(kithttp.NewServer(
		gapi.ListGroupsEndpoint(svc),
		gapi.DecodeListParentsRequest,
		api.EncodeResponse,
		opts...,
	), "list_parents"))

	mux.Post("/groups/:groupID/enable", otelhttp.NewHandler(kithttp.NewServer(
		gapi.EnableGroupEndpoint(svc),
		gapi.DecodeChangeGroupStatus,
		api.EncodeResponse,
		opts...,
	), "enable_group"))

	mux.Post("/groups/:groupID/disable", otelhttp.NewHandler(kithttp.NewServer(
		gapi.DisableGroupEndpoint(svc),
		gapi.DecodeChangeGroupStatus,
		api.EncodeResponse,
		opts...,
	), "disable_group"))

	return mux
}

// func decodeListMembershipRequest(_ context.Context, r *http.Request) (interface{}, error) {
// 	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefGroupStatus)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	level, err := apiutil.ReadNumQuery[uint64](r, api.LevelKey, api.DefLevel)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	parentID, err := apiutil.ReadStringQuery(r, api.ParentKey, "")
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	ownerID, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	meta, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	dir, err := apiutil.ReadNumQuery[int64](r, api.DirKey, -1)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	st, err := mfclients.ToStatus(s)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	req := listMembershipReq{
// 		token:    apiutil.ExtractBearerToken(r),
// 		clientID: bone.GetValue(r, "userID"),
// 		GroupsPage: mfgroups.GroupsPage{
// 			Level: level,
// 			ID:    parentID,
// 			Page: mfgroups.Page{
// 				Offset:   offset,
// 				Limit:    limit,
// 				OwnerID:  ownerID,
// 				Name:     name,
// 				Metadata: meta,
// 				Status:   st,
// 			},
// 			Direction: dir,
// 		},
// 	}
// 	return req, nil
// }

// func decodeListGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
// 	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefGroupStatus)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	level, err := apiutil.ReadNumQuery[uint64](r, api.LevelKey, api.DefLevel)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	parentID, err := apiutil.ReadStringQuery(r, api.ParentKey, "")
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	ownerID, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	meta, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	tree, err := apiutil.ReadBoolQuery(r, api.TreeKey, false)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	dir, err := apiutil.ReadNumQuery[int64](r, api.DirKey, -1)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	st, err := mfclients.ToStatus(s)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	req := listGroupsReq{
// 		token: apiutil.ExtractBearerToken(r),
// 		tree:  tree,
// 		GroupsPage: mfgroups.GroupsPage{
// 			Level: level,
// 			ID:    parentID,
// 			Page: mfgroups.Page{
// 				Offset:   offset,
// 				Limit:    limit,
// 				OwnerID:  ownerID,
// 				Name:     name,
// 				Metadata: meta,
// 				Status:   st,
// 			},
// 			Direction: dir,
// 		},
// 	}
// 	return req, nil
// }

// func decodeListParentsRequest(_ context.Context, r *http.Request) (interface{}, error) {
// 	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefGroupStatus)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	level, err := apiutil.ReadNumQuery[uint64](r, api.LevelKey, api.DefLevel)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	ownerID, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	meta, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	tree, err := apiutil.ReadBoolQuery(r, api.TreeKey, false)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	st, err := mfclients.ToStatus(s)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	req := listGroupsReq{
// 		token: apiutil.ExtractBearerToken(r),
// 		tree:  tree,
// 		GroupsPage: mfgroups.GroupsPage{
// 			Level: level,
// 			ID:    bone.GetValue(r, "groupID"),
// 			Page: mfgroups.Page{
// 				Offset:   offset,
// 				Limit:    limit,
// 				OwnerID:  ownerID,
// 				Name:     name,
// 				Metadata: meta,
// 				Status:   st,
// 			},
// 			Direction: 1,
// 		},
// 	}
// 	return req, nil
// }

// func decodeListChildrenRequest(_ context.Context, r *http.Request) (interface{}, error) {
// 	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefGroupStatus)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	level, err := apiutil.ReadNumQuery[uint64](r, api.LevelKey, api.DefLevel)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	ownerID, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	meta, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	tree, err := apiutil.ReadBoolQuery(r, api.TreeKey, false)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	st, err := mfclients.ToStatus(s)
// 	if err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, err)
// 	}
// 	req := listGroupsReq{
// 		token: apiutil.ExtractBearerToken(r),
// 		tree:  tree,
// 		GroupsPage: mfgroups.GroupsPage{
// 			Level: level,
// 			ID:    bone.GetValue(r, "groupID"),
// 			Page: mfgroups.Page{
// 				Offset:   offset,
// 				Limit:    limit,
// 				OwnerID:  ownerID,
// 				Name:     name,
// 				Metadata: meta,
// 				Status:   st,
// 			},
// 			Direction: -1,
// 		},
// 	}
// 	return req, nil
// }

// func decodeGroupCreate(_ context.Context, r *http.Request) (interface{}, error) {
// 	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
// 		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
// 	}
// 	var g mfgroups.Group
// 	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
// 	}
// 	req := createGroupReq{
// 		Group: g,
// 		token: apiutil.ExtractBearerToken(r),
// 	}

// 	return req, nil
// }

// func decodeGroupUpdate(_ context.Context, r *http.Request) (interface{}, error) {
// 	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
// 		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
// 	}
// 	req := updateGroupReq{
// 		id:    bone.GetValue(r, "groupID"),
// 		token: apiutil.ExtractBearerToken(r),
// 	}
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
// 	}
// 	return req, nil
// }

// func decodeGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
// 	req := groupReq{
// 		token: apiutil.ExtractBearerToken(r),
// 		id:    bone.GetValue(r, "groupID"),
// 	}
// 	return req, nil
// }

// func decodeChangeGroupStatus(_ context.Context, r *http.Request) (interface{}, error) {
// 	req := changeGroupStatusReq{
// 		token: apiutil.ExtractBearerToken(r),
// 		id:    bone.GetValue(r, "groupID"),
// 	}
// 	return req, nil
// }
