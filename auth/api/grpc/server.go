// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/golang/protobuf/ptypes/empty"
	mainflux "github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ mainflux.UsersAuthServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	mainflux.UnimplementedUsersAuthServiceServer
	issue           kitgrpc.Handler
	login           kitgrpc.Handler
	refresh         kitgrpc.Handler
	identify        kitgrpc.Handler
	authorize       kitgrpc.Handler
	addPolicy       kitgrpc.Handler
	deletePolicy    kitgrpc.Handler
	listObjects     kitgrpc.Handler
	listAllObjects  kitgrpc.Handler
	countObjects    kitgrpc.Handler
	listSubjects    kitgrpc.Handler
	listAllSubjects kitgrpc.Handler
	countSubjects   kitgrpc.Handler
	assign          kitgrpc.Handler
	members         kitgrpc.Handler
}

// NewServer returns new AuthServiceServer instance.
func NewServer(svc auth.Service) mainflux.UsersAuthServiceServer {
	return &grpcServer{
		issue: kitgrpc.NewServer(
			(issueEndpoint(svc)),
			decodeIssueRequest,
			encodeIssueResponse,
		),
		login: kitgrpc.NewServer(
			(loginEndpoint(svc)),
			decodeLoginRequest,
			encodeIssueResponse,
		),
		refresh: kitgrpc.NewServer(
			(refreshEndpoint(svc)),
			decodeRefreshRequest,
			encodeIssueResponse,
		),
		identify: kitgrpc.NewServer(
			(identifyEndpoint(svc)),
			decodeIdentifyRequest,
			encodeIdentifyResponse,
		),
		authorize: kitgrpc.NewServer(
			(authorizeEndpoint(svc)),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
		addPolicy: kitgrpc.NewServer(
			(addPolicyEndpoint(svc)),
			decodeAddPolicyRequest,
			encodeAddPolicyResponse,
		),
		deletePolicy: kitgrpc.NewServer(
			(deletePolicyEndpoint(svc)),
			decodeDeletePolicyRequest,
			encodeDeletePolicyResponse,
		),
		listObjects: kitgrpc.NewServer(
			(listObjectsEndpoint(svc)),
			decodeListObjectsRequest,
			encodeListObjectsResponse,
		),
		listAllObjects: kitgrpc.NewServer(
			(listAllObjectsEndpoint(svc)),
			decodeListObjectsRequest,
			encodeListObjectsResponse,
		),
		countObjects: kitgrpc.NewServer(
			(countObjectsEndpoint(svc)),
			decodeCountObjectsRequest,
			encodeCountObjectsResponse,
		),
		listSubjects: kitgrpc.NewServer(
			(listSubjectsEndpoint(svc)),
			decodeListSubjectsRequest,
			encodeListSubjectsResponse,
		),
		listAllSubjects: kitgrpc.NewServer(
			(listAllSubjectsEndpoint(svc)),
			decodeListSubjectsRequest,
			encodeListSubjectsResponse,
		),
		countSubjects: kitgrpc.NewServer(
			(countSubjectsEndpoint(svc)),
			decodeCountSubjectsRequest,
			encodeCountSubjectsResponse,
		),
		assign: kitgrpc.NewServer(
			(assignEndpoint(svc)),
			decodeAssignRequest,
			encodeEmptyResponse,
		),
		members: kitgrpc.NewServer(
			(membersEndpoint(svc)),
			decodeMembersRequest,
			encodeMembersResponse,
		),
	}
}

func (s *grpcServer) Issue(ctx context.Context, req *mainflux.IssueReq) (*mainflux.Token, error) {
	_, res, err := s.issue.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.Token), nil
}

func (s *grpcServer) Login(ctx context.Context, req *mainflux.LoginReq) (*mainflux.Token, error) {
	_, res, err := s.login.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.Token), nil
}

func (s *grpcServer) Refresh(ctx context.Context, req *mainflux.RefreshReq) (*mainflux.Token, error) {
	_, res, err := s.refresh.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.Token), nil
}

func (s *grpcServer) Identify(ctx context.Context, token *mainflux.Token) (*mainflux.UserIdentity, error) {
	_, res, err := s.identify.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.UserIdentity), nil
}

func (s *grpcServer) Authorize(ctx context.Context, req *mainflux.AuthorizeReq) (*mainflux.AuthorizeRes, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.AuthorizeRes), nil
}

func (s *grpcServer) AddPolicy(ctx context.Context, req *mainflux.AddPolicyReq) (*mainflux.AddPolicyRes, error) {
	_, res, err := s.addPolicy.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.AddPolicyRes), nil
}

func (s *grpcServer) DeletePolicy(ctx context.Context, req *mainflux.DeletePolicyReq) (*mainflux.DeletePolicyRes, error) {
	_, res, err := s.deletePolicy.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.DeletePolicyRes), nil
}

func (s *grpcServer) ListObjects(ctx context.Context, req *mainflux.ListObjectsReq) (*mainflux.ListObjectsRes, error) {
	_, res, err := s.listObjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.ListObjectsRes), nil
}

func (s *grpcServer) ListAllObjects(ctx context.Context, req *mainflux.ListObjectsReq) (*mainflux.ListObjectsRes, error) {
	_, res, err := s.listAllObjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.ListObjectsRes), nil
}

func (s *grpcServer) CountObjects(ctx context.Context, req *mainflux.CountObjectsReq) (*mainflux.CountObjectsRes, error) {
	_, res, err := s.countObjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.CountObjectsRes), nil
}

func (s *grpcServer) ListSubjects(ctx context.Context, req *mainflux.ListSubjectsReq) (*mainflux.ListSubjectsRes, error) {
	_, res, err := s.listSubjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.ListSubjectsRes), nil
}

func (s *grpcServer) ListAllSubjects(ctx context.Context, req *mainflux.ListSubjectsReq) (*mainflux.ListSubjectsRes, error) {
	_, res, err := s.listAllSubjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.ListSubjectsRes), nil
}

func (s *grpcServer) CountSubjects(ctx context.Context, req *mainflux.CountSubjectsReq) (*mainflux.CountSubjectsRes, error) {
	_, res, err := s.countSubjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.CountSubjectsRes), nil
}

func (s *grpcServer) Assign(ctx context.Context, token *mainflux.Assignment) (*empty.Empty, error) {
	_, res, err := s.assign.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*empty.Empty), nil
}

func (s *grpcServer) Members(ctx context.Context, req *mainflux.MembersReq) (*mainflux.MembersRes, error) {
	_, res, err := s.members.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*mainflux.MembersRes), nil
}

func decodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.IssueReq)
	return issueReq{id: req.GetId(), email: req.GetEmail(), keyType: auth.KeyType(req.GetType())}, nil
}

func decodeLoginRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.LoginReq)
	return issueReq{id: req.GetId(), email: req.GetEmail(), keyType: auth.AccessKey}, nil
}

func decodeRefreshRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.RefreshReq)
	return refreshReq{value: req.GetValue()}, nil
}

func encodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(issueRes)
	fields := map[string]*structpb.Value{}
	for k, v := range res.extra {
		val, err := structpb.NewValue(v)
		if err != nil {
			return nil, err
		}
		fields[k] = val
	}

	extra := &structpb.Struct{
		Fields: fields,
	}
	return &mainflux.Token{Value: res.value, Extra: extra}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.Token)
	return identityReq{token: req.GetValue()}, nil
}

func encodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identityRes)
	return &mainflux.UserIdentity{Id: res.id, Email: res.email}, nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.AuthorizeReq)
	return authReq{Namespace: req.GetNamespace(),
		SubjectType: req.GetSubjectType(),
		SubjectKind: req.GetSubjectKind(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject()}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authorizeRes)
	return &mainflux.AuthorizeRes{Authorized: res.authorized, Id: res.id}, nil
}

func decodeAddPolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.AddPolicyReq)
	return policyReq{Namespace: req.GetNamespace(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject()}, nil
}

func encodeAddPolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(addPolicyRes)
	return &mainflux.AddPolicyRes{Authorized: res.authorized}, nil
}

func decodeAssignRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.Token)
	return assignReq{token: req.GetValue()}, nil
}

func decodeDeletePolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.DeletePolicyReq)
	return policyReq{Namespace: req.GetNamespace(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject()}, nil
}

func encodeDeletePolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(deletePolicyRes)
	return &mainflux.DeletePolicyRes{Deleted: res.deleted}, nil
}

func decodeListObjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.ListObjectsReq)
	return listObjectsReq{Namespace: req.GetNamespace(),
		SubjectType:   req.GetSubjectType(),
		Subject:       req.GetSubject(),
		Relation:      req.GetRelation(),
		Permission:    req.GetPermission(),
		ObjectType:    req.GetObjectType(),
		Object:        req.GetObject(),
		NextPageToken: req.GetNextPageToken(),
		Limit:         req.GetLimit()}, nil
}

func encodeListObjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(listObjectsRes)
	return &mainflux.ListObjectsRes{Policies: res.policies}, nil
}

func decodeCountObjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.CountObjectsReq)
	return countObjectsReq{Namespace: req.GetNamespace(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject()}, nil
}

func encodeCountObjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(countObjectsRes)
	return &mainflux.CountObjectsRes{Count: int64(res.count)}, nil
}

func decodeListSubjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.ListSubjectsReq)
	return listSubjectsReq{Namespace: req.GetNamespace(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject(), NextPageToken: req.GetNextPageToken(), Limit: req.GetLimit()}, nil
}

func encodeListSubjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(listSubjectsRes)
	return &mainflux.ListSubjectsRes{Policies: res.policies}, nil
}

func decodeCountSubjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.CountSubjectsReq)
	return countSubjectsReq{Namespace: req.GetNamespace(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject()}, nil
}

func encodeCountSubjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(countObjectsRes)
	return &mainflux.CountObjectsRes{Count: int64(res.count)}, nil
}

func decodeMembersRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.MembersReq)
	return membersReq{
		token:      req.GetToken(),
		groupID:    req.GetGroupID(),
		memberType: req.GetType(),
		offset:     req.Offset,
		limit:      req.Limit,
	}, nil
}

func encodeMembersResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(membersRes)
	return &mainflux.MembersRes{
		Total:   res.total,
		Offset:  res.offset,
		Limit:   res.limit,
		Type:    res.groupType,
		Members: res.members,
	}, nil
}

func encodeEmptyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(emptyRes)
	return &empty.Empty{}, encodeError(res.err)
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrInvalidAuthKey,
		err == apiutil.ErrMissingID,
		err == apiutil.ErrMissingMemberType,
		err == apiutil.ErrMissingPolicySub,
		err == apiutil.ErrMissingPolicyObj,
		err == apiutil.ErrMalformedPolicyAct:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication),
		errors.Contains(err, auth.ErrKeyExpired),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
