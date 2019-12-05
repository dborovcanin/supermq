// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/users"
	"google.golang.org/grpc"
)

var _ mainflux.AuthnServiceClient = (*authnServiceMock)(nil)

type authnServiceMock struct {
	users map[string]string
}

// NewAuthService creates mock of users service.
func NewAuthService(users map[string]string) mainflux.AuthnServiceClient {
	return &authnServiceMock{users}
}

func (svc authnServiceMock) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserID, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &mainflux.UserID{Value: id}, nil
	}
	return nil, users.ErrUnauthorizedAccess
}

func (svc authnServiceMock) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	if id, ok := svc.users[in.GetIssuer()]; ok {
		switch in.Type {
		default:
			return &mainflux.Token{Value: id}, nil
		}
	}
	return nil, users.ErrUnauthorizedAccess
}