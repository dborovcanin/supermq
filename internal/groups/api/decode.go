package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux/internal/api"
	"github.com/mainflux/mainflux/internal/apiutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
)

func DecodeListMembershipRequest(_ context.Context, r *http.Request) (interface{}, error) {
	pm, err := decodePageMeta(r)
	if err != nil {
		return nil, err
	}

	level, err := apiutil.ReadNumQuery[uint64](r, api.LevelKey, api.DefLevel)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	parentID, err := apiutil.ReadStringQuery(r, api.ParentKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	dir, err := apiutil.ReadNumQuery[int64](r, api.DirKey, -1)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listMembershipReq{
		token:    apiutil.ExtractBearerToken(r),
		clientID: bone.GetValue(r, "userID"),
		Page: mfgroups.Page{
			Level:     level,
			ID:        parentID,
			PageMeta:  pm,
			Direction: dir,
		},
	}
	return req, nil
}

func DecodeListGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	pm, err := decodePageMeta(r)
	if err != nil {
		return nil, err
	}

	level, err := apiutil.ReadNumQuery[uint64](r, api.LevelKey, api.DefLevel)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	parentID, err := apiutil.ReadStringQuery(r, api.ParentKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	tree, err := apiutil.ReadBoolQuery(r, api.TreeKey, false)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	dir, err := apiutil.ReadNumQuery[int64](r, api.DirKey, -1)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listGroupsReq{
		token: apiutil.ExtractBearerToken(r),
		tree:  tree,
		Page: mfgroups.Page{
			Level:     level,
			ID:        parentID,
			PageMeta:  pm,
			Direction: dir,
		},
	}
	return req, nil
}

func DecodeListParentsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	pm, err := decodePageMeta(r)
	if err != nil {
		return nil, err
	}

	level, err := apiutil.ReadNumQuery[uint64](r, api.LevelKey, api.DefLevel)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	tree, err := apiutil.ReadBoolQuery(r, api.TreeKey, false)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listGroupsReq{
		token: apiutil.ExtractBearerToken(r),
		tree:  tree,
		Page: mfgroups.Page{
			Level:     level,
			ID:        bone.GetValue(r, "groupID"),
			PageMeta:  pm,
			Direction: 1,
		},
	}
	return req, nil
}

func DecodeListChildrenRequest(_ context.Context, r *http.Request) (interface{}, error) {
	pm, err := decodePageMeta(r)
	if err != nil {
		return nil, err
	}

	level, err := apiutil.ReadNumQuery[uint64](r, api.LevelKey, api.DefLevel)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	tree, err := apiutil.ReadBoolQuery(r, api.TreeKey, false)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listGroupsReq{
		token: apiutil.ExtractBearerToken(r),
		tree:  tree,
		Page: mfgroups.Page{
			Level:     level,
			ID:        bone.GetValue(r, "groupID"),
			PageMeta:  pm,
			Direction: -1,
		},
	}
	return req, nil
}

func DecodeGroupCreate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	var g mfgroups.Group
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	req := createGroupReq{
		Group: g,
		token: apiutil.ExtractBearerToken(r),
	}

	return req, nil
}

func DecodeGroupUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := updateGroupReq{
		id:    bone.GetValue(r, "groupID"),
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	return req, nil
}

func DecodeGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := groupReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "groupID"),
	}
	return req, nil
}

func DecodeChangeGroupStatus(_ context.Context, r *http.Request) (interface{}, error) {
	req := changeGroupStatusReq{
		token: apiutil.ExtractBearerToken(r),
		id:    bone.GetValue(r, "groupID"),
	}
	return req, nil
}

func decodePageMeta(r *http.Request) (mfgroups.PageMeta, error) {
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefGroupStatus)
	if err != nil {
		return mfgroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	st, err := mfclients.ToStatus(s)
	if err != nil {
		return mfgroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return mfgroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return mfgroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	ownerID, err := apiutil.ReadStringQuery(r, api.OwnerKey, "")
	if err != nil {
		return mfgroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return mfgroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	meta, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return mfgroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	ret := mfgroups.PageMeta{
		Offset:   offset,
		Limit:    limit,
		Name:     name,
		OwnerID:  ownerID,
		Metadata: meta,
		Status:   st,
	}
	return ret, nil
}
