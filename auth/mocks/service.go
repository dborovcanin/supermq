package mocks

import (
	context "context"

	"github.com/mainflux/mainflux"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ mainflux.AuthServiceClient = (*Service)(nil)

type Service struct {
	mock.Mock
}

func (m *Service) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.Token), ret.Error(1)
}

func (m *Service) Login(ctx context.Context, in *mainflux.LoginReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.Token), ret.Error(1)
}

func (m *Service) Refresh(ctx context.Context, in *mainflux.RefreshReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.Token), ret.Error(1)
}

func (m *Service) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.UserIdentity), ret.Error(1)
}

func (m *Service) Authorize(ctx context.Context, in *mainflux.AuthorizeReq, opts ...grpc.CallOption) (*mainflux.AuthorizeRes, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.AuthorizeRes), ret.Error(1)
}

func (m *Service) AddPolicy(ctx context.Context, in *mainflux.AddPolicyReq, opts ...grpc.CallOption) (*mainflux.AddPolicyRes, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.AddPolicyRes), ret.Error(1)
}

func (m *Service) DeletePolicy(ctx context.Context, in *mainflux.DeletePolicyReq, opts ...grpc.CallOption) (*mainflux.DeletePolicyRes, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.DeletePolicyRes), ret.Error(1)
}

func (m *Service) ListObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.ListObjectsRes), ret.Error(1)
}

func (m *Service) ListAllObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.ListObjectsRes), ret.Error(1)
}

func (m *Service) CountObjects(ctx context.Context, in *mainflux.CountObjectsReq, opts ...grpc.CallOption) (*mainflux.CountObjectsRes, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.CountObjectsRes), ret.Error(1)
}

func (m *Service) ListSubjects(ctx context.Context, in *mainflux.ListSubjectsReq, opts ...grpc.CallOption) (*mainflux.ListSubjectsRes, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.ListSubjectsRes), ret.Error(1)
}

func (m *Service) ListAllSubjects(ctx context.Context, in *mainflux.ListSubjectsReq, opts ...grpc.CallOption) (*mainflux.ListSubjectsRes, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.ListSubjectsRes), ret.Error(1)
}

func (m *Service) CountSubjects(ctx context.Context, in *mainflux.CountSubjectsReq, opts ...grpc.CallOption) (*mainflux.CountSubjectsRes, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.CountSubjectsRes), ret.Error(1)
}

func (m *Service) Assign(ctx context.Context, in *mainflux.Assignment, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*emptypb.Empty), ret.Error(1)
}

func (m *Service) Members(ctx context.Context, in *mainflux.MembersReq, opts ...grpc.CallOption) (*mainflux.MembersRes, error) {
	ret := m.Called(ctx, in)

	return ret.Get(0).(*mainflux.MembersRes), ret.Error(1)
}
