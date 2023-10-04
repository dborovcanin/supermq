// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"fmt"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/apiutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/groups"
)

// Possible token types are access and refresh tokens.
const (
	RefreshToken = "refresh"
	AccessToken  = "access"

	MyKey = "mine"

	groupsObjectKey = "groups"

	updateRelationKey = "g_update"
	listRelationKey   = "g_list"
	deleteRelationKey = "g_delete"
)

const (
	ownerRelation   = "owner"
	channelRelation = "channel"

	usersKind    = "users"
	groupsKind   = "groups"
	thingsKind   = "things"
	channelsKind = "channels"

	userType    = "user"
	groupType   = "group"
	thingType   = "thing"
	channelType = "channel"

	adminPermission      = "admin"
	ownerPermission      = "delete"
	deletePermission     = "delete"
	sharePermission      = "share"
	editPermission       = "edit"
	disconnectPermission = "disconnect"
	connectPermission    = "connect"
	viewPermission       = "view"
	memberPermission     = "member"

	tokenKind = "token"
)

type service struct {
	groups     groups.Repository
	auth       mainflux.AuthServiceClient
	idProvider mainflux.IDProvider
}

// NewService returns a new Clients service implementation.
func NewService(g groups.Repository, idp mainflux.IDProvider, auth mainflux.AuthServiceClient) groups.Service {
	return service{
		groups:     g,
		idProvider: idp,
		auth:       auth,
	}
}

func (svc service) CreateGroup(ctx context.Context, token string, g groups.Group) (groups.Group, error) {
	ownerID, err := svc.identify(ctx, token)
	if err != nil {
		return groups.Group{}, err
	}
	groupID, err := svc.idProvider.ID()
	if err != nil {
		return groups.Group{}, err
	}
	if g.Status != mfclients.EnabledStatus && g.Status != mfclients.DisabledStatus {
		return groups.Group{}, apiutil.ErrInvalidStatus
	}
	if g.Owner == "" {
		g.Owner = ownerID
	}

	g.ID = groupID
	g.CreatedAt = time.Now()

	g, err = svc.groups.Save(ctx, g)
	if err != nil {
		return groups.Group{}, err
	}

	policy := mainflux.AddPolicyReq{
		SubjectType: userType,
		Subject:     ownerID,
		Relation:    ownerRelation,
		ObjectType:  groupType,
		Object:      g.ID,
	}
	if _, err := svc.auth.AddPolicy(ctx, &policy); err != nil {
		return groups.Group{}, err
	}
	return g, nil
}

func (svc service) ViewGroup(ctx context.Context, token string, id string) (groups.Group, error) {
	_, err := svc.authorize(ctx, userType, token, viewPermission, groupType, id)
	if err != nil {
		return groups.Group{}, err
	}

	return svc.groups.RetrieveByID(ctx, id)
}

func (svc service) ListGroups(ctx context.Context, token string, gm groups.Page) (groups.Page, error) {
	id, err := svc.identify(ctx, token)
	if err != nil {
		return groups.Page{}, err
	}

	// If the user is admin, fetch all groups from the database.
	if err := svc.authorizeByID(ctx, id, groupsObjectKey, listRelationKey); err == nil {
		return svc.groups.RetrieveAll(ctx, gm)
	}

	gm.Subject = id
	gm.OwnerID = id
	gm.Action = listRelationKey
	return svc.groups.RetrieveAll(ctx, gm)
}

func (svc service) ListMemberships(ctx context.Context, token, clientID string, gm groups.Page) (groups.Memberships, error) {
	id, err := svc.identify(ctx, token)
	if err != nil {
		return groups.Memberships{}, err
	}
	// If the user is admin, fetch all members from the database.
	if err := svc.authorizeByID(ctx, id, groupsObjectKey, listRelationKey); err == nil {
		return svc.groups.Memberships(ctx, clientID, gm)
	}

	gm.Subject = id
	gm.Action = listRelationKey
	return svc.groups.Memberships(ctx, clientID, gm)
}

func (svc service) UpdateGroup(ctx context.Context, token string, g groups.Group) (groups.Group, error) {
	id, err := svc.authorize(ctx, userType, token, editPermission, groupType, g.ID)
	if err != nil {
		return groups.Group{}, err
	}

	g.UpdatedAt = time.Now()
	g.UpdatedBy = id

	return svc.groups.Update(ctx, g)
}

func (svc service) EnableGroup(ctx context.Context, token, id string) (groups.Group, error) {
	group := groups.Group{
		ID:        id,
		Status:    mfclients.EnabledStatus,
		UpdatedAt: time.Now(),
	}
	group, err := svc.changeGroupStatus(ctx, token, group)
	if err != nil {
		return groups.Group{}, err
	}
	return group, nil
}

func (svc service) DisableGroup(ctx context.Context, token, id string) (groups.Group, error) {
	group := groups.Group{
		ID:        id,
		Status:    mfclients.DisabledStatus,
		UpdatedAt: time.Now(),
	}
	group, err := svc.changeGroupStatus(ctx, token, group)
	if err != nil {
		return groups.Group{}, err
	}
	return group, nil
}

// Yet to do
func (svc service) Assign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) error {
	_, err := svc.authorize(ctx, userType, token, editPermission, groupType, groupID)
	if err != nil {
		return err
	}

	if err := svc.groups.Assign(ctx, groupID, memberKind, memberIDs...); err != nil {
		return err
	}

	prs := []*mainflux.AddPolicyReq{}
	switch memberKind {
	case thingsKind:
		for _, memberID := range memberIDs {
			prs = append(prs, &mainflux.AddPolicyReq{
				SubjectType: groupType,
				Subject:     groupID,
				Relation:    relation,
				ObjectType:  thingType,
				Object:      memberID,
			})
		}
	case channelsKind:
		for _, memberID := range memberIDs {
			prs = append(prs, &mainflux.AddPolicyReq{
				SubjectType: groupType,
				Subject:     groupID,
				Relation:    relation,
				ObjectType:  channelType,
				Object:      memberID,
			})
		}
	case groupsKind:
		for _, memberID := range memberIDs {
			prs = append(prs, &mainflux.AddPolicyReq{
				SubjectType: groupType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  groupType,
				Object:      groupID,
			})
		}
	case usersKind:
		for _, memberID := range memberIDs {
			prs = append(prs, &mainflux.AddPolicyReq{
				SubjectType: userType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  groupType,
				Object:      groupID,
			})
		}
	default:
		for _, memberID := range memberIDs {
			prs = append(prs, &mainflux.AddPolicyReq{
				SubjectType: userType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  groupType,
				Object:      groupID,
			})
		}
	}

	for _, pr := range prs {
		if _, err := svc.auth.AddPolicy(ctx, pr); err != nil {
			return fmt.Errorf("failed to add policies : %w", err)
		}
	}
	return nil
}

// Yet to do
func (svc service) Unassign(ctx context.Context, token, groupID string, memberIDs ...string) error {
	_, err := svc.authorize(ctx, userType, token, editPermission, groupType, groupID)
	if err != nil {
		return err
	}

	prs := []*mainflux.DeletePolicyReq{}
	for _, memberID := range memberIDs {

		prs = append(prs, &mainflux.DeletePolicyReq{
			SubjectType: userType,
			Subject:     memberID,
			ObjectType:  groupType,
			Object:      groupID,
		})
		//  member is thing - same logic is used in previous code, so followed same, not to break api
		prs = append(prs, &mainflux.DeletePolicyReq{
			SubjectType: thingType,
			Subject:     memberID,
			ObjectType:  groupType,
			Object:      groupID,
		})
		//  member is channel - same logic is used in previous code, so followed same, not to break api
		prs = append(prs, &mainflux.DeletePolicyReq{
			SubjectType: channelType,
			Subject:     memberID,
			ObjectType:  groupType,
			Object:      groupID,
		})

		//  member is group - same logic is used in previous code, so followed same, not to break api
		prs = append(prs, &mainflux.DeletePolicyReq{
			SubjectType: groupType,
			Subject:     memberID,
			ObjectType:  groupType,
			Object:      groupID,
		})
	}

	for _, pr := range prs {
		if _, err := svc.auth.DeletePolicy(ctx, pr); err != nil {
			return fmt.Errorf("failed to delete policies : %w", err)
		}
	}
	if err := svc.groups.Unassign(ctx, groupID, memberIDs...); err != nil {
		return err
	}
	return nil
}

func (svc service) changeGroupStatus(ctx context.Context, token string, group groups.Group) (groups.Group, error) {
	id, err := svc.authorize(ctx, userType, token, editPermission, groupType, group.ID)
	if err != nil {
		return groups.Group{}, err
	}
	dbGroup, err := svc.groups.RetrieveByID(ctx, group.ID)
	if err != nil {
		return groups.Group{}, err
	}
	if dbGroup.Status == group.Status {
		return groups.Group{}, mfclients.ErrStatusAlreadyAssigned
	}

	group.UpdatedBy = id
	return svc.groups.ChangeStatus(ctx, group)
}

func (svc service) authorizeByID(ctx context.Context, subject, object, action string) error {
	// policy := policies.Policy{Subject: subject, Object: object, Actions: []string{action}}
	// if err := policy.Validate(); err != nil {
	// 	return err
	// }
	// if err := svc.policies.CheckAdmin(ctx, policy.Subject); err == nil {
	// 	return nil
	// }
	// aReq := policies.AccessRequest{Subject: subject, Object: object, Action: action}
	// if _, err := svc.policies.EvaluateGroupAccess(ctx, aReq); err != nil {
	// 	return err
	// }
	return nil
}

func (svc service) identify(ctx context.Context, token string) (string, error) {
	user, err := svc.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return "", err
	}
	return user.GetId(), nil
}

func (svc service) authorize(ctx context.Context, subjectType, subject, permission, objectType, object string) (string, error) {
	req := &mainflux.AuthorizeReq{
		SubjectType: subjectType,
		SubjectKind: tokenKind,
		Subject:     subject,
		Permission:  permission,
		Object:      object,
		ObjectType:  objectType,
	}
	res, err := svc.auth.Authorize(ctx, req)
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return "", errors.ErrAuthorization
	}
	return res.GetId(), nil
}
