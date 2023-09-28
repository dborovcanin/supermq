// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ mainflux.AuthServiceClient = (*authServiceMock)(nil)

type SubjectSet struct {
	Subject  string
	Relation []string
}

type authServiceMock struct {
	users map[string]string
	authz map[string][]SubjectSet
}

// NewAuthService creates mock of users service.
func NewAuthService(users map[string]string, authzDB map[string][]SubjectSet) mainflux.AuthServiceClient {
	return &authServiceMock{users, authzDB}
}

func (svc authServiceMock) Identify(ctx context.Context, req *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	if id, ok := svc.users[req.Value]; ok {
		return &mainflux.UserIdentity{Id: id, Email: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	if id, ok := svc.users[in.GetEmail()]; ok {
		switch in.Type {
		default:
			return &mainflux.Token{Value: id}, nil
		}
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *mainflux.AuthorizeReq, _ ...grpc.CallOption) (r *mainflux.AuthorizeRes, err error) {
	panic("not implemented")
}

func (svc authServiceMock) AddPolicy(ctx context.Context, in *mainflux.AddPolicyReq, opts ...grpc.CallOption) (*mainflux.AddPolicyRes, error) {
	panic("not implemented")
}
func (svc authServiceMock) DeletePolicy(ctx context.Context, in *mainflux.DeletePolicyReq, opts ...grpc.CallOption) (*mainflux.DeletePolicyRes, error) {
	panic("not implemented")
}
func (svc authServiceMock) ListObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	panic("not implemented")
}
func (svc authServiceMock) ListAllObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	panic("not implemented")
}
func (svc authServiceMock) CountObjects(ctx context.Context, in *mainflux.CountObjectsReq, opts ...grpc.CallOption) (*mainflux.CountObjectsRes, error) {
	panic("not implemented")
}
func (svc authServiceMock) ListSubjects(ctx context.Context, in *mainflux.ListSubjectsReq, opts ...grpc.CallOption) (*mainflux.ListSubjectsRes, error) {
	panic("not implemented")
}
func (svc authServiceMock) ListAllSubjects(ctx context.Context, in *mainflux.ListSubjectsReq, opts ...grpc.CallOption) (*mainflux.ListSubjectsRes, error) {
	panic("not implemented")
}
func (svc authServiceMock) CountSubjects(ctx context.Context, in *mainflux.CountSubjectsReq, opts ...grpc.CallOption) (*mainflux.CountSubjectsRes, error) {
	panic("not implemented")
}
func (svc authServiceMock) Assign(ctx context.Context, in *mainflux.Assignment, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	panic("not implemented")
}
func (svc authServiceMock) Members(ctx context.Context, in *mainflux.MembersReq, opts ...grpc.CallOption) (*mainflux.MembersRes, error) {
	panic("not implemented")
}
