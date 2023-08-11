// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"google.golang.org/grpc"
)

var _ mainflux.AuthServiceClient = (*authServiceMock)(nil)

type MockSubjectSet struct {
	Object   string
	Relation string
}

type authServiceMock struct {
	users    map[string]string
	policies map[string][]MockSubjectSet
}

func (svc authServiceMock) ListObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	res := mainflux.ListObjectsRes{}
	for key := range svc.policies {
		res.Policies = append(res.Policies, key)
	}
	return &res, nil
}

func (svc authServiceMock) ListAllObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	res := mainflux.ListObjectsRes{}
	for key := range svc.policies {
		res.Policies = append(res.Policies, key)
	}
	return &res, nil
}

// NewAuthService creates mock of users service.
func NewAuthService(users map[string]string, policies map[string][]MockSubjectSet) mainflux.AuthServiceClient {
	return &authServiceMock{users, policies}
}

func (svc authServiceMock) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	if id, ok := svc.users[in.Value]; ok {
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
	for _, policy := range svc.policies[req.GetSubject()] {
		if policy.Relation == req.GetPermission() && policy.Object == req.GetObject() {
			return &mainflux.AuthorizeRes{Authorized: true}, nil
		}
	}
	return nil, errors.ErrAuthorization
}

func (svc authServiceMock) AddPolicy(ctx context.Context, in *mainflux.AddPolicyReq, opts ...grpc.CallOption) (*mainflux.AddPolicyRes, error) {
	if in.GetPermission() == "" || in.GetObject() == "" || in.GetSubject() == "" {
		return &mainflux.AddPolicyRes{}, errors.ErrMalformedEntity
	}

	obj := in.GetObject()
	svc.policies[in.GetSubject()] = append(svc.policies[in.GetSubject()], MockSubjectSet{Object: obj, Relation: in.GetPermission()})
	return &mainflux.AddPolicyRes{Authorized: true}, nil
}

func (svc authServiceMock) DeletePolicy(ctx context.Context, in *mainflux.DeletePolicyReq, opts ...grpc.CallOption) (*mainflux.DeletePolicyRes, error) {
	// Not implemented yet
	return &mainflux.DeletePolicyRes{Deleted: true}, nil
}

func (svc authServiceMock) Members(ctx context.Context, req *mainflux.MembersReq, _ ...grpc.CallOption) (r *mainflux.MembersRes, err error) {
	panic("not implemented")
}

func (svc authServiceMock) Assign(ctx context.Context, req *mainflux.Assignment, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	panic("not implemented")
}

func (svc authServiceMock) CountObjects(ctx context.Context, req *mainflux.CountObjectsReq, _ ...grpc.CallOption) (r *mainflux.CountObjectsRes, err error) {
	panic("not implemented")
}

func (svc authServiceMock) ListSubjects(ctx context.Context, req *mainflux.ListSubjectsReq, _ ...grpc.CallOption) (r *mainflux.ListSubjectsRes, err error) {
	panic("not implemented")
}
func (svc authServiceMock) ListAllSubjects(ctx context.Context, req *mainflux.ListSubjectsReq, _ ...grpc.CallOption) (r *mainflux.ListSubjectsRes, err error) {
	panic("not implemented")
}
func (svc authServiceMock) CountSubjects(ctx context.Context, req *mainflux.CountSubjectsReq, _ ...grpc.CallOption) (r *mainflux.CountSubjectsRes, err error) {
	panic("not implemented")
}
