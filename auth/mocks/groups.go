// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/stretchr/testify/mock"
)

const WrongID = "wrongID"

var _ auth.GroupRepository = (*Repository)(nil)

type Repository struct {
	mock.Mock
}

func (m *Repository) ChangeStatus(ctx context.Context, group auth.Group) (auth.Group, error) {
	ret := m.Called(ctx, group)

	if group.ID == WrongID {
		return auth.Group{}, errors.ErrNotFound
	}

	return ret.Get(0).(auth.Group), ret.Error(1)
}

func (m *Repository) RetrieveByIDs(ctx context.Context, groupIDs []string, gm auth.PageMetadata) (auth.GroupPage, error) {
	ret := m.Called(ctx, groupIDs, gm)

	return ret.Get(0).(auth.GroupPage), ret.Error(1)
}

func (m *Repository) MembershipsByGroupIDs(ctx context.Context, groupIDs []string, memberID string, pm auth.PageMetadata) (auth.GroupPage, error) {
	ret := m.Called(ctx, groupIDs, memberID, pm)

	return ret.Get(0).(auth.GroupPage), ret.Error(1)
}

func (m *Repository) RetrieveAll(ctx context.Context, gm auth.PageMetadata) (auth.GroupPage, error) {
	ret := m.Called(ctx, gm)

	return ret.Get(0).(auth.GroupPage), ret.Error(1)
}

func (m *Repository) RetrieveAllChildren(ctx context.Context, groupID string, gm auth.PageMetadata) (auth.GroupPage, error) {
	ret := m.Called(ctx, groupID, gm)

	return ret.Get(0).(auth.GroupPage), ret.Error(1)
}

func (m *Repository) RetrieveAllParents(ctx context.Context, groupID string, gm auth.PageMetadata) (auth.GroupPage, error) {
	ret := m.Called(ctx, groupID, gm)

	return ret.Get(0).(auth.GroupPage), ret.Error(1)
}

func (m *Repository) RetrieveByID(ctx context.Context, id string) (auth.Group, error) {
	ret := m.Called(ctx, id)

	if id == WrongID {
		return auth.Group{}, errors.ErrNotFound
	}

	return ret.Get(0).(auth.Group), ret.Error(1)
}

func (m *Repository) Save(ctx context.Context, g auth.Group) (auth.Group, error) {
	ret := m.Called(ctx, g)

	return g, ret.Error(1)
}

func (m *Repository) Update(ctx context.Context, g auth.Group) (auth.Group, error) {
	ret := m.Called(ctx, g)

	if g.ID == WrongID {
		return auth.Group{}, errors.ErrNotFound
	}

	return ret.Get(0).(auth.Group), ret.Error(1)
}

func (m *Repository) Delete(ctx context.Context, id string) error {
	ret := m.Called(ctx, id)

	if id == WrongID {
		return errors.ErrNotFound
	}

	return ret.Error(0)
}

func (m *Repository) Assign(ctx context.Context, groupID, memberType string, memberIDs ...string) error {
	ret := m.Called(ctx, groupID, memberType, memberIDs)

	if groupID == WrongID {
		return errors.ErrNotFound
	}

	return ret.Error(0)
}

func (m *Repository) Unassign(ctx context.Context, groupID string, memberIDs ...string) error {
	ret := m.Called(ctx, groupID, memberIDs)

	if groupID == WrongID {
		return errors.ErrNotFound
	}

	return ret.Error(0)
}

func (m *Repository) Members(ctx context.Context, groupID, memberType string, pm auth.PageMetadata) (auth.MemberPage, error) {
	ret := m.Called(ctx, groupID, memberType, pm)

	if groupID == WrongID {
		return auth.MemberPage{}, errors.ErrNotFound
	}

	return ret.Get(0).(auth.MemberPage), ret.Error(1)
}

func (m *Repository) Memberships(ctx context.Context, memberID string, pm auth.PageMetadata) (auth.GroupPage, error) {
	ret := m.Called(ctx, memberID, pm)

	if memberID == WrongID {
		return auth.GroupPage{}, errors.ErrNotFound
	}

	return ret.Get(0).(auth.GroupPage), ret.Error(1)
}
