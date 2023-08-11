// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"
	"fmt"
	"sync"

	"github.com/mainflux/mainflux/pkg/errors"
	"google.golang.org/grpc"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/ulid"
)

const (
	administratorRelationKey = "administrator"
	directMemberRelation     = "direct_member"
	ownerRelation            = "owner"
	editorRelation           = "owner"
	viewerRelation           = "viewer"
	organizationRelation     = "organization"
	groupRelation            = "group"
	channelRelation          = "channel"

	adminPermission      = "admin"
	ownerPermission      = "delete"
	deletePermission     = "delete"
	sharePermission      = "share"
	editPermission       = "edit"
	disconnectPermission = "disconnect"
	connectPermission    = "connect"
	viewPermission       = "view"
	memberPermission     = "member"

	userType         = "user"
	organizationType = "organization"
	thingType        = "thing"
	channelType      = "channel"

	mainfluxObject = "mainflux"
	anyBodySubject = "_any_body"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// CreateThings adds things to the user identified by the provided key.
	CreateThings(ctx context.Context, token string, things ...Thing) ([]Thing, error)

	// UpdateThing updates the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateThing(ctx context.Context, token string, thing Thing) error

	// ShareThing gives actions associated with the thing to the given user IDs.
	// The requester user identified by the token has to have a "write" relation
	// on the thing in order to share the thing.
	ShareThing(ctx context.Context, token, thingID string, actions, userIDs []string) error

	// UpdateKey updates key value of the existing thing. A non-nil error is
	// returned to indicate operation failure.
	UpdateKey(ctx context.Context, token, id, key string) error

	// ViewThing retrieves data about the thing identified with the provided
	// ID, that belongs to the user identified by the provided key.
	ViewThing(ctx context.Context, token, id string) (Thing, error)

	// ListThings retrieves data about subset of things that belongs to the
	// user identified by the provided key.
	ListThings(ctx context.Context, token string, pm PageMetadata) (Page, error)

	// ListThingsByChannel retrieves data about subset of things that are
	// connected or not connected to specified channel and belong to the user identified by
	// the provided key.
	ListThingsByChannel(ctx context.Context, token, chID string, pm PageMetadata) (Page, error)

	// RemoveThing removes the thing identified with the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveThing(ctx context.Context, token, id string) error

	// CreateChannels adds channels to the user identified by the provided key.
	CreateChannels(ctx context.Context, token string, channels ...Channel) ([]Channel, error)

	// UpdateChannel updates the channel identified by the provided ID, that
	// belongs to the user identified by the provided key.
	UpdateChannel(ctx context.Context, token string, channel Channel) error

	// ViewChannel retrieves data about the channel identified by the provided
	// ID, that belongs to the user identified by the provided key.
	ViewChannel(ctx context.Context, token, id string) (Channel, error)

	// ShareChannel gives actions associated with the channel to the given user IDs.
	// The requester user identified by the token has to have a "write" relation
	// on the thing in order to share the channel.
	ShareChannel(ctx context.Context, token, thingID string, actions, userIDs []string) error

	// ListChannels retrieves data about subset of channels that belongs to the
	// user identified by the provided key.
	ListChannels(ctx context.Context, token string, pm PageMetadata) (ChannelsPage, error)

	// ListChannelsByThing retrieves data about subset of channels that have
	// specified thing connected or not connected to them and belong to the user identified by
	// the provided key.
	ListChannelsByThing(ctx context.Context, token, thID string, pm PageMetadata) (ChannelsPage, error)

	// RemoveChannel removes the thing identified by the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveChannel(ctx context.Context, token, id string) error

	// Connect adds things to the channels list of connected things.
	Connect(ctx context.Context, token string, chIDs, thIDs []string) error

	// Disconnect removes things from the channels list of connected
	// things.
	Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error

	// CanAccessByKey determines whether the channel can be accessed using the
	// provided key and returns thing's id if access is allowed.
	CanAccessByKey(ctx context.Context, chanID, key string) (string, error)

	// CanAccessByID determines whether the channel can be accessed by
	// the given thing and returns error if it cannot.
	CanAccessByID(ctx context.Context, chanID, thingID string) error

	// IsChannelOwner determines whether the channel can be accessed by
	// the given user and returns error if it cannot.
	IsChannelOwner(ctx context.Context, owner, chanID string) error

	// Identify returns thing ID for given thing key.
	Identify(ctx context.Context, key string) (string, error)

	// ListThingMembers retrieves every things that is assigned to a group identified by groupID.
	ListThingMembers(ctx context.Context, token, groupID string, pm PageMetadata) (Page, error)

	// ListChannelMembers retrieves every things that is assigned to a group identified by groupID.
	ListChannelMembers(ctx context.Context, token, groupID string, pm PageMetadata) (ChannelsPage, error)
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total             uint64
	Offset            uint64                 `json:"offset,omitempty"`
	Limit             uint64                 `json:"limit,omitempty"`
	Name              string                 `json:"name,omitempty"`
	Order             string                 `json:"order,omitempty"`
	Dir               string                 `json:"dir,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	Disconnected      bool                   // Used for connected or disconnected lists
	FetchSharedThings bool                   // Used for identifying fetching either all or shared things.
}

var _ Service = (*thingsService)(nil)

type thingsService struct {
	auth         mainflux.AuthServiceClient
	things       ThingRepository
	channels     ChannelRepository
	channelCache ChannelCache
	thingCache   ThingCache
	idProvider   mainflux.IDProvider
	ulidProvider mainflux.IDProvider
}

// New instantiates the things service implementation.
func New(auth mainflux.AuthServiceClient, things ThingRepository, channels ChannelRepository, ccache ChannelCache, tcache ThingCache, idp mainflux.IDProvider) Service {
	return &thingsService{
		auth:         auth,
		things:       things,
		channels:     channels,
		channelCache: ccache,
		thingCache:   tcache,
		idProvider:   idp,
		ulidProvider: ulid.New(),
	}
}

func (ts *thingsService) CreateThings(ctx context.Context, token string, things ...Thing) ([]Thing, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Thing{}, err
	}

	if err := ts.authorize(ctx, userType, res.GetId(), memberPermission, organizationType, mainfluxObject); err != nil {
		return []Thing{}, err
	}

	ths := []Thing{}
	for _, thing := range things {
		th, err := ts.createThing(ctx, &thing, res)

		if err != nil {
			return []Thing{}, err
		}
		ths = append(ths, th)
	}

	return ths, nil
}

// createThing saves the Thing and adds identity as an owner(Read, Write, Delete policies) of the Thing.
func (ts *thingsService) createThing(ctx context.Context, thing *Thing, identity *mainflux.UserIdentity) (Thing, error) {

	thing.Owner = identity.GetEmail()

	if thing.ID == "" {
		id, err := ts.idProvider.ID()
		if err != nil {
			return Thing{}, err
		}
		thing.ID = id
	}

	if thing.Key == "" {
		key, err := ts.idProvider.ID()

		if err != nil {
			return Thing{}, err
		}
		thing.Key = key
	}

	ths, err := ts.things.Save(ctx, *thing)
	if err != nil {
		return Thing{}, err
	}
	if len(ths) == 0 {
		return Thing{}, errors.ErrCreateEntity
	}

	policy := mainflux.AddPolicyReq{
		SubjectType: userType,
		Subject:     identity.GetId(),
		Relation:    ownerRelation,
		ObjectType:  thingType,
		Object:      ths[0].ID,
	}
	if err := ts.AddPolicy(ctx, &policy); err != nil {
		return Thing{}, err
	}

	return ths[0], nil
}

func (ts *thingsService) UpdateThing(ctx context.Context, token string, thing Thing) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, userType, res.GetId(), editPermission, thingType, thing.ID); err != nil {
		return err
	}

	thing.Owner = res.GetEmail()

	return ts.things.Update(ctx, thing)
}

func (ts *thingsService) ShareThing(ctx context.Context, token, thingID string, relations, userIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, userType, res.GetId(), sharePermission, thingType, thingID); err != nil {
		return err
	}

	for _, id := range userIDs {
		if err := ts.authorize(ctx, userType, id, memberPermission, organizationRelation, mainfluxObject); err != nil {
			return fmt.Errorf("failed to authorize user id %s : %w", id, err)
		}
	}

	var policies []*mainflux.AddPolicyReq

	for _, id := range userIDs {
		for _, relation := range relations {
			policies = append(policies, &mainflux.AddPolicyReq{
				SubjectType: userType,
				Subject:     id,
				Relation:    relation,
				ObjectType:  thingType,
				Object:      thingID,
			})
		}

	}
	return ts.AddPolicies(ctx, policies)
}

func (ts *thingsService) AddPolicies(ctx context.Context, policies []*mainflux.AddPolicyReq) error {
	var errs error
	for _, policy := range policies {
		err := ts.AddPolicy(ctx, policy)
		errs = errors.Wrap(errs, err)
	}
	return errs
}

func (ts *thingsService) AddPolicy(ctx context.Context, policy *mainflux.AddPolicyReq) error {
	apr, err := ts.auth.AddPolicy(ctx, policy)
	if err != nil {
		return fmt.Errorf("cannot add policy sub:'%s:%s' relation:'%s' obj:'%s:%s' error:'%w' ", policy.SubjectType, policy.Subject, policy.Relation, policy.ObjectType, policy.Object, err)
	}
	if !apr.GetAuthorized() {
		return fmt.Errorf("cannot add policy sub:'%s:%s' relation:'%s' obj:'%s:%s' error:'unauthorized' ", policy.SubjectType, policy.Subject, policy.Relation, policy.ObjectType, policy.Object)
	}
	return nil
}

func (ts *thingsService) UpdateKey(ctx context.Context, token, id, key string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, userType, res.GetId(), editPermission, thingType, id); err != nil {
		return err
	}

	owner := res.GetEmail()

	return ts.things.UpdateKey(ctx, owner, id, key)
}

func (ts *thingsService) ViewThing(ctx context.Context, token, id string) (Thing, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Thing{}, err
	}

	if err := ts.authorize(ctx, userType, res.GetId(), viewPermission, thingType, id); err != nil {
		return Thing{}, err
	}

	return ts.things.RetrieveByID(ctx, res.GetEmail(), id)
}

func (ts *thingsService) ListThings(ctx context.Context, token string, pm PageMetadata) (Page, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, err
	}

	ids := make(map[string]struct{})

	req := &mainflux.ListObjectsReq{
		SubjectType: userType,
		Subject:     res.GetId(),
		Permission:  viewPermission,
		ObjectType:  thingType,
	}

	lpr, err := ts.auth.ListAllObjects(ctx, req, grpc.MaxCallRecvMsgSize(5120000000))
	if err != nil {
		return Page{}, err
	}

	for _, policy := range lpr.Policies { // List of Things ID
		ids[policy] = struct{}{}
	}

	thingIds := []string{}
	for id := range ids {
		thingIds = append(thingIds, id)
	}
	page, err := ts.things.RetrieveByIDs(ctx, thingIds, pm)
	if err != nil {
		return Page{}, err
	}

	return page, nil
}

func (ts *thingsService) ListThingsByChannel(ctx context.Context, token, chID string, pm PageMetadata) (Page, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, err
	}
	if err := ts.authorize(ctx, userType, res.GetId(), viewPermission, channelType, chID); err != nil {
		return Page{}, err
	}
	req := &mainflux.ListObjectsReq{
		SubjectType: channelType,
		Subject:     chID,
		Permission:  viewPermission,
		ObjectType:  thingType,
	}
	lpr, err := ts.auth.ListAllObjects(ctx, req)
	if err != nil {
		return Page{}, err
	}
	allowedThingIDs := []string{}
	for _, policy := range lpr.Policies {
		var wg sync.WaitGroup
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if err := ts.authorize(ctx, userType, res.GetId(), viewPermission, thingType, policy); err == nil {
				allowedThingIDs = append(allowedThingIDs, policy)
			}
		}(&wg)
		wg.Wait()
	}

	switch pm.Disconnected {
	case true:
		req2 := &mainflux.ListObjectsReq{
			SubjectType: userType,
			Subject:     res.Id,
			Permission:  connectPermission,
			ObjectType:  thingType,
		}
		lpr2, err := ts.auth.ListAllObjects(ctx, req2)
		if err != nil {
			return Page{}, err
		}

		disconnectedThings := map[string]struct{}{}

		for _, thingid := range lpr2.Policies {
			for _, connThingID := range allowedThingIDs {
				if thingid != connThingID {
					disconnectedThings[thingid] = struct{}{}
				}
			}
		}

		disThingID := []string{}
		for tid := range disconnectedThings {
			disThingID = append(disThingID, tid)
		}
		return ts.things.RetrieveByIDs(ctx, disThingID, pm)
	default:
		return ts.things.RetrieveByIDs(ctx, allowedThingIDs, pm)
	}
}

func (ts *thingsService) RemoveThing(ctx context.Context, token, id string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, userType, res.GetId(), deletePermission, thingType, id); err != nil {
		return err
	}

	if err := ts.thingCache.Remove(ctx, id); err != nil {
		return err
	}
	return ts.things.Remove(ctx, res.GetEmail(), id)
}

func (ts *thingsService) CreateChannels(ctx context.Context, token string, channels ...Channel) ([]Channel, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return []Channel{}, err
	}

	chs := []Channel{}
	for _, channel := range channels {
		ch, err := ts.createChannel(ctx, &channel, res)
		if err != nil {
			return []Channel{}, err
		}
		chs = append(chs, ch)
	}
	return chs, nil
}

func (ts *thingsService) createChannel(ctx context.Context, channel *Channel, identity *mainflux.UserIdentity) (Channel, error) {
	if channel.ID == "" {
		chID, err := ts.idProvider.ID()
		if err != nil {
			return Channel{}, err
		}
		channel.ID = chID
	}
	channel.Owner = identity.GetEmail()

	chs, err := ts.channels.Save(ctx, *channel)
	if err != nil {
		return Channel{}, err
	}
	if len(chs) == 0 {
		return Channel{}, errors.ErrCreateEntity
	}

	policy := mainflux.AddPolicyReq{
		SubjectType: userType,
		Subject:     identity.GetId(),
		Relation:    ownerRelation,
		ObjectType:  channelType,
		Object:      chs[0].ID,
	}
	if err := ts.AddPolicy(ctx, &policy); err != nil {
		return Channel{}, err
	}
	return chs[0], nil
}

func (ts *thingsService) UpdateChannel(ctx context.Context, token string, channel Channel) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, userType, res.GetId(), editPermission, channelType, channel.ID); err != nil {
		return err
	}

	channel.Owner = res.GetEmail()
	return ts.channels.Update(ctx, channel)
}

func (ts *thingsService) ShareChannel(ctx context.Context, token, channelID string, relations, userIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, userType, res.GetId(), sharePermission, channelType, channelID); err != nil {
		return err
	}

	for _, id := range userIDs {
		if err := ts.authorize(ctx, userType, id, memberPermission, organizationRelation, mainfluxObject); err != nil {
			return fmt.Errorf("failed to authorize user id %s : %w", id, err)
		}
	}

	var policies []*mainflux.AddPolicyReq

	for _, id := range userIDs {
		for _, relation := range relations {
			policies = append(policies, &mainflux.AddPolicyReq{
				SubjectType: userType,
				Subject:     id,
				Relation:    relation,
				ObjectType:  channelType,
				Object:      channelID,
			})
		}

	}
	return ts.AddPolicies(ctx, policies)
}

func (ts *thingsService) ViewChannel(ctx context.Context, token, id string) (Channel, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Channel{}, err
	}

	if err := ts.authorize(ctx, userType, res.GetId(), viewPermission, channelType, id); err != nil {
		return Channel{}, err
	}

	return ts.channels.RetrieveByID(ctx, res.GetEmail(), id)
}

func (ts *thingsService) ListChannels(ctx context.Context, token string, pm PageMetadata) (ChannelsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, err
	}

	req := &mainflux.ListObjectsReq{
		SubjectType: userType,
		Subject:     res.GetId(),
		Permission:  viewPermission,
		ObjectType:  channelType,
	}
	lpr, err := ts.auth.ListAllObjects(ctx, req)
	if err != nil {
		return ChannelsPage{}, err
	}

	chPage, err := ts.channels.RetrieveByIDs(ctx, lpr.Policies, pm)
	if err != nil {
		return ChannelsPage{}, err
	}

	// By default, fetch channels from database based on the owner field.
	return chPage, nil
}

func (ts *thingsService) ListChannelsByThing(ctx context.Context, token, thID string, pm PageMetadata) (ChannelsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ChannelsPage{}, err
	}

	if err := ts.authorize(ctx, userType, res.GetId(), viewPermission, thingType, thID); err != nil {
		return ChannelsPage{}, err
	}

	req := &mainflux.ListSubjectsReq{
		SubjectType: channelType,
		Permission:  viewPermission,
		ObjectType:  thingType,
		Object:      thID,
	}

	lpr, err := ts.auth.ListAllSubjects(ctx, req)
	if err != nil {
		return ChannelsPage{}, err
	}

	allowedChannelIDs := []string{}
	for _, policy := range lpr.Policies {
		var wg sync.WaitGroup
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			if err := ts.authorize(ctx, userType, res.GetId(), viewPermission, channelType, policy); err == nil {
				allowedChannelIDs = append(allowedChannelIDs, policy)
			}
		}(&wg)
		wg.Wait()
	}

	switch pm.Disconnected {
	case true:
		req2 := &mainflux.ListSubjectsReq{
			SubjectType: channelType,
			Permission:  viewPermission,
			ObjectType:  thingType,
			Object:      thID,
		}

		lpr2, err := ts.auth.ListAllSubjects(ctx, req2)
		if err != nil {
			return ChannelsPage{}, err
		}

		disconnectedChannelIDs := map[string]struct{}{}

		for _, cid := range lpr2.Policies {
			for _, connCid := range allowedChannelIDs {
				if cid != connCid {
					disconnectedChannelIDs[cid] = struct{}{}
				}
			}
		}

		disChanIDs := []string{}

		for cid := range disconnectedChannelIDs {
			disChanIDs = append(disChanIDs, cid)
		}
		return ts.channels.RetrieveByIDs(ctx, disChanIDs, pm)

	default:
		return ts.channels.RetrieveByIDs(ctx, allowedChannelIDs, pm)
	}

}

func (ts *thingsService) RemoveChannel(ctx context.Context, token, id string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	if err := ts.authorize(ctx, userType, res.GetId(), deletePermission, channelType, id); err != nil {
		return err
	}

	if err := ts.channelCache.Remove(ctx, id); err != nil {
		return err
	}

	return ts.channels.Remove(ctx, res.GetEmail(), id)
}

func (ts *thingsService) Connect(ctx context.Context, token string, chIDs, thIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	for _, thID := range thIDs {
		if err := ts.authorize(ctx, userType, res.GetId(), connectPermission, thingType, thID); err != nil {
			return fmt.Errorf("failed to authorize for thing id %s : %w", thID, err)
		}
	}

	for _, chID := range chIDs {
		if err := ts.authorize(ctx, userType, res.GetId(), connectPermission, channelType, chID); err != nil {
			return fmt.Errorf("failed to authorize for channel id %s : %w", chID, err)
		}
	}

	// Operation tries to be atomic, So the previous loops are not used
	policies := []*mainflux.AddPolicyReq{}
	for _, chID := range chIDs {
		for _, thID := range thIDs {
			policies = append(policies, &mainflux.AddPolicyReq{
				SubjectType: channelType,
				Subject:     chID,
				Relation:    channelRelation,
				ObjectType:  thingType,
				Object:      thID,
			})
		}
	}

	if err := ts.AddPolicies(ctx, policies); err != nil {
		return fmt.Errorf("failed to add policies : %w", err)
	}
	return nil
	// return ts.channels.Connect(ctx, res.GetEmail(), chIDs, thIDs)
}

func (ts *thingsService) Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return err
	}

	for _, thID := range thIDs {
		if err := ts.authorize(ctx, userType, res.GetId(), disconnectPermission, thingType, thID); err != nil {
			return err
		}
	}

	for _, chID := range chIDs {
		if err := ts.authorize(ctx, userType, res.GetId(), disconnectPermission, channelType, chID); err != nil {
			return err
		}
	}

	// Operation tries to be atomic, So the previous loops are not used
	for _, chID := range chIDs {
		for _, thID := range thIDs {
			_, err := ts.auth.DeletePolicy(ctx, &mainflux.DeletePolicyReq{
				SubjectType: channelType,
				Subject:     chID,
				Relation:    channelRelation,
				ObjectType:  thingType,
				Object:      thID,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil

	// return ts.channels.Disconnect(ctx, res.GetEmail(), chIDs, thIDs)
}

func (ts *thingsService) CanAccessByKey(ctx context.Context, chanID, thingKey string) (string, error) {
	thingID, err := ts.hasThing(ctx, chanID, thingKey)
	if err == nil {
		return thingID, nil
	}

	thingID, err = ts.things.RetrieveByKey(ctx, thingKey)
	if err != nil {
		return "", err
	}

	if err := ts.authorize(ctx, channelType, chanID, viewPermission, thingType, thingID); err != nil {
		return "", err
	}

	if err := ts.thingCache.Save(ctx, thingKey, thingID); err != nil {
		return "", err
	}
	if err := ts.channelCache.Connect(ctx, chanID, thingID); err != nil {
		return "", err
	}
	return thingID, nil
}

func (ts *thingsService) CanAccessByID(ctx context.Context, chanID, thingID string) error {
	if connected := ts.channelCache.HasThing(ctx, chanID, thingID); connected {
		return nil
	}

	if err := ts.authorize(ctx, channelType, chanID, viewPermission, thingType, thingID); err != nil {
		return err
	}

	if err := ts.channelCache.Connect(ctx, chanID, thingID); err != nil {
		return err
	}
	return nil
}

func (ts *thingsService) IsChannelOwner(ctx context.Context, userID, chanID string) error {
	if err := ts.authorize(ctx, userType, userID, ownerPermission, channelType, chanID); err != nil {
		return err
	}
	return nil
}

func (ts *thingsService) Identify(ctx context.Context, key string) (string, error) {
	id, err := ts.thingCache.ID(ctx, key)
	if err == nil {
		return id, nil
	}

	id, err = ts.things.RetrieveByKey(ctx, key)
	if err != nil {
		return "", err
	}

	if err := ts.thingCache.Save(ctx, key, id); err != nil {
		return "", err
	}
	return id, nil
}

func (ts *thingsService) hasThing(ctx context.Context, chanID, thingKey string) (string, error) {
	thingID, err := ts.thingCache.ID(ctx, thingKey)
	if err != nil {
		return "", err
	}

	if connected := ts.channelCache.HasThing(ctx, chanID, thingID); !connected {
		return "", errors.ErrAuthorization
	}
	return thingID, nil
}

func (ts *thingsService) ListThingMembers(ctx context.Context, token, groupID string, pm PageMetadata) (Page, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return Page{}, err
	}

	res, err := ts.members(ctx, token, groupID, "things", pm.Offset, pm.Limit)
	if err != nil {
		return Page{}, nil
	}

	return ts.things.RetrieveByIDs(ctx, res, pm)
}

func (ts *thingsService) ListChannelMembers(ctx context.Context, token, groupID string, pm PageMetadata) (ChannelsPage, error) {
	if _, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token}); err != nil {
		return ChannelsPage{}, err
	}

	res, err := ts.members(ctx, token, groupID, "channels", pm.Offset, pm.Limit)
	if err != nil {
		return ChannelsPage{}, nil
	}

	return ts.channels.RetrieveByIDs(ctx, res, pm)
}

func (ts *thingsService) members(ctx context.Context, token, groupID, groupType string, limit, offset uint64) ([]string, error) {
	req := mainflux.MembersReq{
		Token:   token,
		GroupID: groupID,
		Offset:  offset,
		Limit:   limit,
		Type:    groupType,
	}

	res, err := ts.auth.Members(ctx, &req)
	if err != nil {
		return nil, nil
	}
	return res.Members, nil
}

func (ts *thingsService) authorize(ctx context.Context, subjectType, subject, permission, objectType, object string) error {
	req := &mainflux.AuthorizeReq{
		SubjectType: subjectType,
		Subject:     subject,
		Permission:  permission,
		Object:      object,
		ObjectType:  objectType,
	}
	res, err := ts.auth.Authorize(ctx, req)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	return nil
}
