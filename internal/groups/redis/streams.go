// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/go-redis/redis/v8"
	mfredis "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/pkg/groups"
)

const (
	streamID  = "mainflux.users"
	streamLen = 1000
)

var _ groups.Service = (*eventStore)(nil)

type eventStore struct {
	mfredis.Publisher
	svc    groups.Service
	client *redis.Client
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc groups.Service, client *redis.Client) groups.Service {
	es := eventStore{
		svc:       svc,
		client:    client,
		Publisher: mfredis.NewEventStore(client, streamID, streamLen),
	}

	go es.StartPublishingRoutine(ctx)

	return es
}

func (es eventStore) CreateGroup(ctx context.Context, token string, group groups.Group) (groups.Group, error) {
	group, err := es.svc.CreateGroup(ctx, token, group)
	if err != nil {
		return group, err
	}

	event := createGroupEvent{
		group,
	}

	if err := es.Publish(ctx, event); err != nil {
		return group, err
	}

	return group, nil
}

func (es eventStore) UpdateGroup(ctx context.Context, token string, group groups.Group) (groups.Group, error) {
	group, err := es.svc.UpdateGroup(ctx, token, group)
	if err != nil {
		return group, err
	}

	event := updateGroupEvent{
		group,
	}

	if err := es.Publish(ctx, event); err != nil {
		return group, err
	}

	return group, nil
}

func (es eventStore) ViewGroup(ctx context.Context, token, id string) (groups.Group, error) {
	group, err := es.svc.ViewGroup(ctx, token, id)
	if err != nil {
		return group, err
	}
	event := viewGroupEvent{
		group,
	}

	if err := es.Publish(ctx, event); err != nil {
		return group, err
	}

	return group, nil
}

func (es eventStore) ListGroups(ctx context.Context, token string, memberKind string, memberID string, pm groups.Page) (groups.Page, error) {
	gp, err := es.svc.ListGroups(ctx, token, memberKind, memberID, pm)
	if err != nil {
		return gp, err
	}
	event := listGroupEvent{
		pm,
	}

	if err := es.Publish(ctx, event); err != nil {
		return gp, err
	}

	return gp, nil
}

func (es eventStore) ListMemberships(ctx context.Context, token, groupID, memberKind string) (groups.Memberships, error) {
	mp, err := es.svc.ListMemberships(ctx, token, groupID, memberKind)
	if err != nil {
		return mp, err
	}
	event := listGroupMembershipEvent{
		groupID, memberKind,
	}

	if err := es.Publish(ctx, event); err != nil {
		return mp, err
	}

	return mp, nil
}

func (es eventStore) EnableGroup(ctx context.Context, token, id string) (groups.Group, error) {
	group, err := es.svc.EnableGroup(ctx, token, id)
	if err != nil {
		return group, err
	}

	return es.delete(ctx, group)
}

func (es eventStore) Assign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) error {
	return es.svc.Assign(ctx, token, groupID, relation, memberKind, memberIDs...)
}

func (es eventStore) Unassign(ctx context.Context, token, groupID string, relation string, memberKind string, memberIDs ...string) error {
	return es.svc.Unassign(ctx, token, groupID, relation, memberKind, memberIDs...)
}

func (es eventStore) DisableGroup(ctx context.Context, token, id string) (groups.Group, error) {
	group, err := es.svc.DisableGroup(ctx, token, id)
	if err != nil {
		return group, err
	}

	return es.delete(ctx, group)
}

func (es eventStore) delete(ctx context.Context, group groups.Group) (groups.Group, error) {
	event := removeGroupEvent{
		id:        group.ID,
		updatedAt: group.UpdatedAt,
		updatedBy: group.UpdatedBy,
		status:    group.Status.String(),
	}

	if err := es.Publish(ctx, event); err != nil {
		return group, err
	}

	return group, nil
}
