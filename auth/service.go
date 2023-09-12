// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/ulid"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	recoveryDuration = 5 * time.Minute
	thingsKind       = "things"
	channelsKind     = "channels"
	usersKind        = "users"

	thingType   = "thing"
	channelType = "channel"
	userType    = "user"
	groupType   = "group"

	memberRelation        = "member"
	groupRelation         = "group"
	administratorRelation = "administrator"
	parentGroupRelation   = "parent_group"
	viewerRelation        = "viewer"

	adminPermission = "admin"
	editPermission  = "edit"
	viewPermission  = "view"

	mainfluxObject = "mainflux"
)

// Possible token types are access and refresh tokens.
const (
	RefreshToken = 0
	AccessToken  = 1
)

var (
	// ErrFailedToRetrieveMembers failed to retrieve group members.
	ErrFailedToRetrieveMembers = errors.New("failed to retrieve group members")

	// ErrFailedToRetrieveMembership failed to retrieve memberships
	ErrFailedToRetrieveMembership = errors.New("failed to retrieve memberships")

	// ErrFailedToRetrieveAll failed to retrieve groups.
	ErrFailedToRetrieveAll = errors.New("failed to retrieve all groups")

	// ErrFailedToRetrieveParents failed to retrieve groups.
	ErrFailedToRetrieveParents = errors.New("failed to retrieve all groups")

	// ErrFailedToRetrieveChildren failed to retrieve groups.
	ErrFailedToRetrieveChildren = errors.New("failed to retrieve all groups")

	errIssueUser = errors.New("failed to issue new login key")
	errIssueTmp  = errors.New("failed to issue new temporary key")
	errRevoke    = errors.New("failed to remove key")
	errRetrieve  = errors.New("failed to retrieve key data")
	errIdentify  = errors.New("failed to validate token")
)

// Authn specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// Token is a string value of the actual Key and is used to authenticate
// an Auth service request.
type Authn interface {
	// Issue issues a new Key, returning its token value alongside.
	Issue(ctx context.Context, token string, key Key) (*mainflux.Token, error)

	// Revoke removes the Key with the provided id that is
	// issued by the user identified by the provided key.
	Revoke(ctx context.Context, token, id string) error

	// RetrieveKey retrieves data for the Key identified by the provided
	// ID, that is issued by the user identified by the provided key.
	RetrieveKey(ctx context.Context, token, id string) (Key, error)

	// Identify validates token token. If token is valid, content
	// is returned. If token is invalid, or invocation failed for some
	// other reason, non-nil error value is returned in response.
	Identify(ctx context.Context, token string) (Identity, error)
}

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// Token is a string value of the actual Key and is used to authenticate
// an Auth service request.
type Service interface {
	Authn
	Authz

	// GroupService implements groups API, creating groups, assigning members
	GroupService
}

var _ Service = (*service)(nil)

type service struct {
	keys            KeyRepository
	groups          GroupRepository
	idProvider      mainflux.IDProvider
	ulidProvider    mainflux.IDProvider
	agent           PolicyAgent
	tokenizer       Tokenizer
	loginDuration   time.Duration
	refreshDuration time.Duration
}

// New instantiates the auth service implementation.
func New(keys KeyRepository, groups GroupRepository, idp mainflux.IDProvider, tokenizer Tokenizer, policyAgent PolicyAgent, duration time.Duration) Service {
	return &service{
		tokenizer:     tokenizer,
		keys:          keys,
		groups:        groups,
		idProvider:    idp,
		ulidProvider:  ulid.New(),
		agent:         policyAgent,
		loginDuration: duration,
	}
}

func (svc service) Issue(ctx context.Context, token string, key Key) (*mainflux.Token, error) {
	if key.IssuedAt.IsZero() {
		return nil, ErrInvalidKeyIssuedAt
	}
	switch key.Type {
	case APIKey:
		return svc.userKey(ctx, token, key)
	case RecoveryKey:
		return svc.tmpKey(recoveryDuration, key)
	default:
		ret, err := svc.accessKey(key)
		return ret, err
	}
}

func (svc service) Revoke(ctx context.Context, token, id string) error {
	issuerID, _, err := svc.login(token)
	if err != nil {
		return errors.Wrap(errRevoke, err)
	}
	if err := svc.keys.Remove(ctx, issuerID, id); err != nil {
		return errors.Wrap(errRevoke, err)
	}
	return nil
}

func (svc service) RetrieveKey(ctx context.Context, token, id string) (Key, error) {
	issuerID, _, err := svc.login(token)
	if err != nil {
		return Key{}, errors.Wrap(errRetrieve, err)
	}

	return svc.keys.Retrieve(ctx, issuerID, id)
}

func (svc service) Identify(ctx context.Context, token string) (Identity, error) {
	key, err := svc.tokenizer.Parse(token)
	if err == ErrAPIKeyExpired {
		err = svc.keys.Remove(ctx, key.IssuerID, key.ID)
		return Identity{}, errors.Wrap(ErrAPIKeyExpired, err)
	}
	if err != nil {
		return Identity{}, errors.Wrap(errIdentify, err)
	}

	switch key.Type {
	case RecoveryKey, AccessKey:
		return Identity{ID: key.IssuerID, Email: key.Subject}, nil
	case APIKey:
		_, err := svc.keys.Retrieve(context.TODO(), key.IssuerID, key.ID)
		if err != nil {
			return Identity{}, errors.ErrAuthentication
		}
		return Identity{ID: key.IssuerID, Email: key.Subject}, nil
	default:
		return Identity{}, errors.ErrAuthentication
	}
}

func (svc service) Authorize(ctx context.Context, pr PolicyReq) error {
	if err := svc.agent.CheckPolicy(ctx, pr); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	return nil
}

func (svc service) AddPolicy(ctx context.Context, pr PolicyReq) error {
	return svc.agent.AddPolicy(ctx, pr)
}

// Yet to do
func (svc service) AddPolicies(ctx context.Context, token, object string, subjectIDs, relations []string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.Authorize(ctx, PolicyReq{Object: mainfluxObject, Subject: user.ID}); err != nil {
		return err
	}

	var errs error
	for _, subjectID := range subjectIDs {
		for _, relation := range relations {
			if err := svc.AddPolicy(ctx, PolicyReq{Object: object, Relation: relation, Subject: subjectID}); err != nil {
				errs = errors.Wrap(fmt.Errorf("cannot add '%s' policy on object '%s' for subject '%s': %s", relation, object, subjectID, err), errs)
			}
		}
	}
	return errs
}

func (svc service) DeletePolicy(ctx context.Context, pr PolicyReq) error {
	return svc.agent.DeletePolicy(ctx, pr)
}

// Yet to do
func (svc service) DeletePolicies(ctx context.Context, token, object string, subjectIDs, relations []string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	// Check if the user identified by token is the admin.
	if err := svc.Authorize(ctx, PolicyReq{Object: mainfluxObject, Subject: user.ID}); err != nil {
		return err
	}

	var errs error
	for _, subjectID := range subjectIDs {
		for _, relation := range relations {
			if err := svc.DeletePolicy(ctx, PolicyReq{Object: object, Relation: relation, Subject: subjectID}); err != nil {
				errs = errors.Wrap(fmt.Errorf("cannot delete '%s' policy on object '%s' for subject '%s': %s", relation, object, subjectID, err), errs)
			}
		}
	}
	return errs
}

func (svc service) AssignGroupAccessRights(ctx context.Context, token, thingGroupID, userGroupID string) error {
	if _, err := svc.Identify(ctx, token); err != nil {
		return err
	}
	return svc.agent.AddPolicy(ctx, PolicyReq{SubjectType: groupType, Subject: userGroupID, Relation: groupRelation, ObjectType: groupType, Object: thingGroupID})
}

func (svc service) ListObjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit int32) (PolicyPage, error) {
	if limit <= 0 {
		limit = 100
	}
	res, npt, err := svc.agent.RetrieveObjects(ctx, pr, nextPageToken, limit)
	if err != nil {
		return PolicyPage{}, err
	}
	var page PolicyPage
	for _, tuple := range res {
		page.Policies = append(page.Policies, tuple.Object)
	}
	page.NextPageToken = npt
	return page, err
}

func (svc service) ListAllObjects(ctx context.Context, pr PolicyReq) (PolicyPage, error) {
	res, err := svc.agent.RetrieveAllObjects(ctx, pr)
	if err != nil {
		return PolicyPage{}, err
	}
	var page PolicyPage
	for _, tuple := range res {
		page.Policies = append(page.Policies, tuple.Object)
	}
	return page, err
}

func (svc service) CountObjects(ctx context.Context, pr PolicyReq) (int, error) {
	return svc.agent.RetrieveAllObjectsCount(ctx, pr)
}

func (svc service) ListSubjects(ctx context.Context, pr PolicyReq, nextPageToken string, limit int32) (PolicyPage, error) {
	if limit <= 0 {
		limit = 100
	}
	res, npt, err := svc.agent.RetrieveSubjects(ctx, pr, nextPageToken, limit)
	if err != nil {
		return PolicyPage{}, err
	}
	var page PolicyPage
	for _, tuple := range res {
		page.Policies = append(page.Policies, tuple.Subject)
	}
	page.NextPageToken = npt
	return page, err
}

func (svc service) ListAllSubjects(ctx context.Context, pr PolicyReq) (PolicyPage, error) {
	res, err := svc.agent.RetrieveAllSubjects(ctx, pr)
	if err != nil {
		return PolicyPage{}, err
	}
	var page PolicyPage
	for _, tuple := range res {
		page.Policies = append(page.Policies, tuple.Subject)
	}
	return page, err
}

func (svc service) CountSubjects(ctx context.Context, pr PolicyReq) (int, error) {
	return svc.agent.RetrieveAllSubjectsCount(ctx, pr)
}

func (svc service) tmpKey(duration time.Duration, key Key) (*mainflux.Token, error) {
	secret, err := svc.tokenizer.Issue(key)
	if err != nil {
		return nil, errors.Wrap(errIssueTmp, err)
	}

	return &mainflux.Token{Value: secret}, nil
}

func (svc service) accessKey(key Key) (*mainflux.Token, error) {
	key.Type = AccessToken
	key.ExpiresAt = time.Now().Add(svc.loginDuration)
	access, err := svc.tokenizer.Issue(key)
	if err != nil {
		return nil, errors.Wrap(errIssueTmp, err)
	}
	key.ExpiresAt = time.Now().Add(svc.refreshDuration)
	key.Type = RefreshToken
	refresh, err := svc.tokenizer.Issue(key)
	if err != nil {
		return nil, errors.Wrap(errIssueTmp, err)
	}
	rfrsh := structpb.NewStringValue(refresh)
	extra := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"refresh_token": rfrsh,
		},
	}
	// fmt.Println("extra", extra.GetStringValue())
	// fmt.Println("extra struct", extra.GetStructValue())
	return &mainflux.Token{Value: access, Extra: extra}, nil
}

func (svc service) userKey(ctx context.Context, token string, key Key) (*mainflux.Token, error) {
	id, sub, err := svc.login(token)
	if err != nil {
		return nil, errors.Wrap(errIssueUser, err)
	}

	key.IssuerID = id
	if key.Subject == "" {
		key.Subject = sub
	}

	keyID, err := svc.idProvider.ID()
	if err != nil {
		return nil, errors.Wrap(errIssueUser, err)
	}
	key.ID = keyID

	if _, err := svc.keys.Save(ctx, key); err != nil {
		return nil, errors.Wrap(errIssueUser, err)
	}

	tkn, err := svc.tokenizer.Issue(key)
	if err != nil {
		return nil, errors.Wrap(errIssueUser, err)
	}

	return &mainflux.Token{Value: tkn}, nil
}

func (svc service) login(token string) (string, string, error) {
	key, err := svc.tokenizer.Parse(token)
	if err != nil {
		return "", "", err
	}
	// Only login key token is valid for login.
	if key.Type != AccessKey || key.IssuerID == "" {
		return "", "", errors.ErrAuthentication
	}

	return key.IssuerID, key.Subject, nil
}

// Done
func (svc service) CreateGroup(ctx context.Context, token string, group Group) (Group, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Group{}, err
	}

	ulid, err := svc.ulidProvider.ID()
	if err != nil {
		return Group{}, err
	}

	timestamp := getTimestmap()
	group.UpdatedAt = timestamp
	group.CreatedAt = timestamp

	group.ID = ulid
	group.OwnerID = user.ID

	group, err = svc.groups.Save(ctx, group)
	if err != nil {
		return Group{}, err
	}

	if group.ParentID != "" {
		if err := svc.agent.AddPolicy(ctx, PolicyReq{SubjectType: groupType, Subject: group.ID, Relation: parentGroupRelation, ObjectType: groupType, Object: group.ParentID}); err != nil {
			return Group{}, fmt.Errorf("failed to add policy for parent group : %w", err)
		}
	}
	if err := svc.agent.AddPolicy(ctx, PolicyReq{SubjectType: userType, Subject: user.ID, Relation: administratorRelation, ObjectType: groupType, Object: group.ID}); err != nil {
		return Group{}, err
	}

	return group, nil
}

// Yet to do
func (svc service) ListGroups(ctx context.Context, token string, pm PageMetadata) (GroupPage, error) {
	identity, err := svc.Identify(ctx, token)
	if err != nil {
		return GroupPage{}, err
	}

	req := PolicyReq{
		SubjectType: userType,
		Subject:     identity.ID,
		Permission:  viewPermission,
		ObjectType:  groupType,
	}

	lpr, err := svc.ListAllObjects(ctx, req)
	if err != nil {
		return GroupPage{}, err
	}
	if len(lpr.Policies) <= 0 {
		return GroupPage{}, nil
	}

	return svc.groups.RetrieveByIDs(ctx, lpr.Policies, pm)
}

// Yet to do
func (svc service) ListParents(ctx context.Context, token string, childID string, pm PageMetadata) (GroupPage, error) {
	identity, err := svc.Identify(ctx, token)
	if err != nil {
		return GroupPage{}, err
	}
	if err := svc.agent.CheckPolicy(ctx, PolicyReq{Subject: identity.ID, SubjectType: userType, Permission: viewPermission, ObjectType: groupType, Object: childID}); err != nil {
		return GroupPage{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	groupsPage, err := svc.groups.RetrieveAllParents(ctx, childID, pm)
	if err != nil {
		return GroupPage{}, err
	}

	allowedGroups := []Group{}
	for _, group := range groupsPage.Groups {
		var wg sync.WaitGroup
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if err := svc.Authorize(ctx, PolicyReq{SubjectType: userType, Subject: identity.ID, Permission: viewPermission, ObjectType: groupType, Object: group.ID}); err == nil {
				allowedGroups = append(allowedGroups, group)
			}
		}(&wg)
		wg.Wait()
	}
	groupsPage.Groups = allowedGroups
	return groupsPage, nil
}

// Yet to do
func (svc service) ListChildren(ctx context.Context, token string, parentID string, pm PageMetadata) (GroupPage, error) {
	identity, err := svc.Identify(ctx, token)
	if err != nil {
		return GroupPage{}, err
	}
	if err := svc.agent.CheckPolicy(ctx, PolicyReq{Subject: identity.ID, SubjectType: userType, Permission: viewPermission, ObjectType: groupType, Object: parentID}); err != nil {
		return GroupPage{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	groupsPage, err := svc.groups.RetrieveAllChildren(ctx, parentID, pm)
	if err != nil {
		return GroupPage{}, err
	}

	allowedGroups := []Group{}
	for _, group := range groupsPage.Groups {
		var wg sync.WaitGroup
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if err := svc.Authorize(ctx, PolicyReq{SubjectType: userType, Subject: identity.ID, Permission: viewPermission, ObjectType: groupType, Object: group.ID}); err == nil {
				allowedGroups = append(allowedGroups, group)
			}
		}(&wg)
		wg.Wait()
	}
	groupsPage.Groups = allowedGroups
	return groupsPage, nil
}

// Yet to do
func (svc service) ListMembers(ctx context.Context, token string, groupID, memberKind string, pm PageMetadata) (MemberPage, error) {
	identity, err := svc.Identify(ctx, token)
	if err != nil {
		return MemberPage{}, err
	}
	if err := svc.agent.CheckPolicy(ctx, PolicyReq{Subject: identity.ID, SubjectType: userType, Permission: viewPermission, ObjectType: groupType, Object: groupID}); err != nil {
		return MemberPage{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	mp, err := svc.groups.Members(ctx, groupID, memberKind, pm)
	if err != nil {
		return MemberPage{}, errors.Wrap(ErrFailedToRetrieveMembers, err)
	}
	return mp, nil
}

// Done
func (svc service) RemoveGroup(ctx context.Context, token, id string) error {
	identity, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}
	if err := svc.agent.CheckPolicy(ctx, PolicyReq{SubjectType: userType, Subject: identity.ID, Permission: adminPermission, ObjectType: groupType, Object: id}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	return svc.groups.Delete(ctx, id)
}

// Done
func (svc service) UpdateGroup(ctx context.Context, token string, group Group) (Group, error) {
	identity, err := svc.Identify(ctx, token)
	if err != nil {
		return Group{}, err
	}
	if err := svc.agent.CheckPolicy(ctx, PolicyReq{SubjectType: userType, Subject: identity.ID, Permission: editPermission, ObjectType: groupType, Object: group.ID}); err != nil {
		return Group{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	group.UpdatedAt = getTimestmap()
	return svc.groups.Update(ctx, group)
}

// Done
func (svc service) ViewGroup(ctx context.Context, token, id string) (Group, error) {
	identity, err := svc.Identify(ctx, token)
	if err != nil {
		return Group{}, err
	}
	if err := svc.agent.CheckPolicy(ctx, PolicyReq{SubjectType: userType, Subject: identity.ID, Permission: viewPermission, ObjectType: groupType, Object: id}); err != nil {
		return Group{}, errors.Wrap(errors.ErrAuthorization, err)
	}
	return svc.groups.RetrieveByID(ctx, id)
}

// Yet to do
func (svc service) Assign(ctx context.Context, token string, groupID, memberKind string, memberIDs ...string) error {
	identity, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}
	if err := svc.agent.CheckPolicy(ctx, PolicyReq{Subject: identity.ID, SubjectType: userType, Permission: editPermission, ObjectType: groupType, Object: groupID}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	if err := svc.groups.Assign(ctx, groupID, memberKind, memberIDs...); err != nil {
		return err
	}

	prs := []PolicyReq{}
	switch memberKind {
	case thingsKind:
		for _, memberID := range memberIDs {
			prs = append(prs, PolicyReq{
				SubjectType: groupType,
				Subject:     groupID,
				Relation:    groupRelation,
				ObjectType:  thingType,
				Object:      memberID,
			})
		}
	case channelsKind:
		for _, memberID := range memberIDs {
			prs = append(prs, PolicyReq{
				SubjectType: groupType,
				Subject:     groupID,
				Relation:    groupRelation,
				ObjectType:  channelType,
				Object:      memberID,
			})
		}
	case usersKind:
		for _, memberID := range memberIDs {
			prs = append(prs, PolicyReq{
				SubjectType: userType,
				Subject:     memberID,
				Relation:    viewerRelation,
				ObjectType:  groupType,
				Object:      groupID,
			})
		}
	default:
		for _, memberID := range memberIDs {
			prs = append(prs, PolicyReq{
				SubjectType: userType,
				Subject:     memberID,
				Relation:    viewerRelation,
				ObjectType:  groupType,
				Object:      groupID,
			})
		}
	}

	if err := svc.agent.AddPolicies(ctx, prs); err != nil {
		return fmt.Errorf("failed to add policies : %w", err)
	}
	return nil
}

// Yet to do
func (svc service) Unassign(ctx context.Context, token string, groupID string, memberIDs ...string) error {
	identity, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}
	if err := svc.agent.CheckPolicy(ctx, PolicyReq{Subject: identity.ID, SubjectType: userType, Permission: editPermission, ObjectType: groupType, Object: groupID}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	prs := []PolicyReq{}
	for _, memberID := range memberIDs {
		//  member is user - same logic is used in previous code, so followed same, not to break api
		prs = append(prs, PolicyReq{
			SubjectType: userType,
			Subject:     memberID,
			Relation:    viewerRelation,
			ObjectType:  groupType,
			Object:      groupID,
		})
		//  member is thing - same logic is used in previous code, so followed same, not to break api
		prs = append(prs, PolicyReq{
			SubjectType: groupType,
			Subject:     groupID,
			Relation:    groupRelation,
			ObjectType:  thingType,
			Object:      memberID,
		})
		//  member is channel - same logic is used in previous code, so followed same, not to break api
		prs = append(prs, PolicyReq{
			SubjectType: groupType,
			Subject:     groupID,
			Relation:    groupRelation,
			ObjectType:  channelType,
			Object:      memberID,
		})
	}

	if err := svc.agent.DeletePolicies(ctx, prs); err != nil {
		return fmt.Errorf("failed to delete policies : %w", err)
	}
	if err := svc.groups.Unassign(ctx, groupID, memberIDs...); err != nil {
		return err
	}
	return nil
}

// Yet to do
func (svc service) ListMemberships(ctx context.Context, token string, memberID string, pm PageMetadata) (GroupPage, error) {
	identity, err := svc.Identify(ctx, token)
	if err != nil {
		return GroupPage{}, err
	}
	req := PolicyReq{
		SubjectType: userType,
		Subject:     identity.ID,
		Permission:  viewPermission,
		ObjectType:  groupType,
	}

	lpr, err := svc.ListAllObjects(ctx, req)
	if err != nil {
		return GroupPage{}, err
	}

	return svc.groups.MembershipsByGroupIDs(ctx, lpr.Policies, memberID, pm)
}

func getTimestmap() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}
