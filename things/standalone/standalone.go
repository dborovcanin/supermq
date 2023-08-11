// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"google.golang.org/grpc"
)

var errUnsupported = errors.New("not supported in standalone mode")

var _ mainflux.AuthServiceClient = (*singleUserRepo)(nil)

type singleUserRepo struct {
	email string
	token string
}

// NewAuthService creates single user repository for constrained environments.
func NewAuthService(email, token string) mainflux.AuthServiceClient {
	return singleUserRepo{
		email: email,
		token: token,
	}
}

func (repo singleUserRepo) Issue(ctx context.Context, req *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	if repo.token != req.GetEmail() {
		return nil, errors.ErrAuthentication
	}

	return &mainflux.Token{Value: repo.token}, nil
}

func (repo singleUserRepo) Identify(ctx context.Context, token *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	if repo.token != token.GetValue() {
		return nil, errors.ErrAuthentication
	}

	return &mainflux.UserIdentity{Id: repo.email, Email: repo.email}, nil
}

func (repo singleUserRepo) Authorize(ctx context.Context, req *mainflux.AuthorizeReq, _ ...grpc.CallOption) (r *mainflux.AuthorizeRes, err error) {
	if repo.email != req.Subject {
		return &mainflux.AuthorizeRes{}, errUnsupported
	}
	return &mainflux.AuthorizeRes{Authorized: true}, nil
}

func (repo singleUserRepo) AddPolicy(ctx context.Context, req *mainflux.AddPolicyReq, opts ...grpc.CallOption) (*mainflux.AddPolicyRes, error) {
	if repo.email != req.Subject {
		return &mainflux.AddPolicyRes{}, errUnsupported
	}
	return &mainflux.AddPolicyRes{Authorized: true}, nil
}

func (repo singleUserRepo) DeletePolicy(ctx context.Context, req *mainflux.DeletePolicyReq, opts ...grpc.CallOption) (*mainflux.DeletePolicyRes, error) {
	if repo.email != req.Subject {
		return &mainflux.DeletePolicyRes{}, errUnsupported
	}
	return &mainflux.DeletePolicyRes{Deleted: true}, nil
}

func (repo singleUserRepo) ListObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	return &mainflux.ListObjectsRes{}, errUnsupported
}

func (repo singleUserRepo) ListAllObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	return &mainflux.ListObjectsRes{}, errUnsupported
}

func (repo singleUserRepo) CountObjects(ctx context.Context, in *mainflux.CountObjectsReq, opts ...grpc.CallOption) (*mainflux.CountObjectsRes, error) {
	return &mainflux.CountObjectsRes{}, errUnsupported
}

func (repo singleUserRepo) ListSubjects(ctx context.Context, in *mainflux.ListSubjectsReq, opts ...grpc.CallOption) (*mainflux.ListSubjectsRes, error) {
	return &mainflux.ListSubjectsRes{}, errUnsupported
}

func (repo singleUserRepo) ListAllSubjects(ctx context.Context, in *mainflux.ListSubjectsReq, opts ...grpc.CallOption) (*mainflux.ListSubjectsRes, error) {
	return &mainflux.ListSubjectsRes{}, errUnsupported
}

func (repo singleUserRepo) CountSubjects(ctx context.Context, in *mainflux.CountSubjectsReq, opts ...grpc.CallOption) (*mainflux.CountSubjectsRes, error) {
	return &mainflux.CountSubjectsRes{}, errUnsupported
}

func (repo singleUserRepo) Members(ctx context.Context, req *mainflux.MembersReq, _ ...grpc.CallOption) (r *mainflux.MembersRes, err error) {
	return &mainflux.MembersRes{}, errUnsupported
}

func (repo singleUserRepo) Assign(ctx context.Context, req *mainflux.Assignment, _ ...grpc.CallOption) (r *empty.Empty, err error) {
	return &empty.Empty{}, errUnsupported
}
