// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

<<<<<<<< HEAD:users/groups/events/streams.go
	"github.com/mainflux/mainflux/pkg/events"
	"github.com/mainflux/mainflux/pkg/events/redis"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/users/groups"
========
	"github.com/go-redis/redis/v8"
	mfredis "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/pkg/groups"
>>>>>>>> 9492132bb (Return Auth service):internal/groups/redis/streams.go
)

const streamID = "mainflux.users"

var _ groups.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc groups.Service
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc groups.Service, url string) (groups.Service, error) {
	publisher, err := redis.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	return &eventStore{
		svc:       svc,
		Publisher: publisher,
	}, nil
}

<<<<<<<< HEAD:users/groups/events/streams.go
func (es *eventStore) CreateGroup(ctx context.Context, token string, group mfgroups.Group) (mfgroups.Group, error) {
========
func (es eventStore) CreateGroup(ctx context.Context, token string, group groups.Group) (groups.Group, error) {
>>>>>>>> 9492132bb (Return Auth service):internal/groups/redis/streams.go
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

<<<<<<<< HEAD:users/groups/events/streams.go
func (es *eventStore) UpdateGroup(ctx context.Context, token string, group mfgroups.Group) (mfgroups.Group, error) {
========
func (es eventStore) UpdateGroup(ctx context.Context, token string, group groups.Group) (groups.Group, error) {
>>>>>>>> 9492132bb (Return Auth service):internal/groups/redis/streams.go
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

<<<<<<<< HEAD:users/groups/events/streams.go
func (es *eventStore) ViewGroup(ctx context.Context, token, id string) (mfgroups.Group, error) {
========
func (es eventStore) ViewGroup(ctx context.Context, token, id string) (groups.Group, error) {
>>>>>>>> 9492132bb (Return Auth service):internal/groups/redis/streams.go
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

<<<<<<<< HEAD:users/groups/events/streams.go
func (es *eventStore) ListGroups(ctx context.Context, token string, pm mfgroups.GroupsPage) (mfgroups.GroupsPage, error) {
	gp, err := es.svc.ListGroups(ctx, token, pm)
========
func (es eventStore) ListGroups(ctx context.Context, token string, memberKind string, memberID string, pm groups.Page) (groups.Page, error) {
	gp, err := es.svc.ListGroups(ctx, token, memberKind, memberID, pm)
>>>>>>>> 9492132bb (Return Auth service):internal/groups/redis/streams.go
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

<<<<<<<< HEAD:users/groups/events/streams.go
func (es *eventStore) ListMemberships(ctx context.Context, token, clientID string, pm mfgroups.GroupsPage) (mfgroups.MembershipsPage, error) {
	mp, err := es.svc.ListMemberships(ctx, token, clientID, pm)
========
func (es eventStore) ListMembers(ctx context.Context, token, groupID, permission, memberKind string) (groups.MembersPage, error) {
	mp, err := es.svc.ListMembers(ctx, token, groupID, permission, memberKind)
>>>>>>>> 9492132bb (Return Auth service):internal/groups/redis/streams.go
	if err != nil {
		return mp, err
	}
	event := listGroupMembershipEvent{
		groupID, permission, memberKind,
	}

	if err := es.Publish(ctx, event); err != nil {
		return mp, err
	}

	return mp, nil
}

<<<<<<<< HEAD:users/groups/events/streams.go
func (es *eventStore) EnableGroup(ctx context.Context, token, id string) (mfgroups.Group, error) {
========
func (es eventStore) EnableGroup(ctx context.Context, token, id string) (groups.Group, error) {
>>>>>>>> 9492132bb (Return Auth service):internal/groups/redis/streams.go
	group, err := es.svc.EnableGroup(ctx, token, id)
	if err != nil {
		return group, err
	}

	return es.delete(ctx, group)
}

<<<<<<<< HEAD:users/groups/events/streams.go
func (es *eventStore) DisableGroup(ctx context.Context, token, id string) (mfgroups.Group, error) {
========
func (es eventStore) Assign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) error {
	return es.svc.Assign(ctx, token, groupID, relation, memberKind, memberIDs...)
}

func (es eventStore) Unassign(ctx context.Context, token, groupID string, relation string, memberKind string, memberIDs ...string) error {
	return es.svc.Unassign(ctx, token, groupID, relation, memberKind, memberIDs...)
}

func (es eventStore) DisableGroup(ctx context.Context, token, id string) (groups.Group, error) {
>>>>>>>> 9492132bb (Return Auth service):internal/groups/redis/streams.go
	group, err := es.svc.DisableGroup(ctx, token, id)
	if err != nil {
		return group, err
	}

	return es.delete(ctx, group)
}

<<<<<<<< HEAD:users/groups/events/streams.go
func (es *eventStore) delete(ctx context.Context, group mfgroups.Group) (mfgroups.Group, error) {
========
func (es eventStore) delete(ctx context.Context, group groups.Group) (groups.Group, error) {
>>>>>>>> 9492132bb (Return Auth service):internal/groups/redis/streams.go
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
