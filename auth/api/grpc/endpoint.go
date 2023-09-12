// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/auth"
)

func issueEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(issueReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{
			Type:     req.keyType,
			Subject:  req.email,
			IssuerID: req.id,
			IssuedAt: time.Now().UTC(),
		}

		tkn, err := svc.Issue(ctx, "", key)
		if err != nil {
			return issueRes{}, err
		}

		return issueRes{tkn.Value}, nil
	}
}

func identifyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identityReq)
		if err := req.validate(); err != nil {
			return identityRes{}, err
		}

		id, err := svc.Identify(ctx, req.token)
		if err != nil {
			return identityRes{}, err
		}

		ret := identityRes{
			id:    id.ID,
			email: id.Email,
		}
		return ret, nil
	}
}

func authorizeEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authReq)

		if err := req.validate(); err != nil {
			return authorizeRes{}, err
		}

		err := svc.Authorize(ctx, auth.PolicyReq{
			Namespace:   req.Namespace,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return authorizeRes{}, err
		}
		return authorizeRes{authorized: true}, err
	}
}

func addPolicyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(policyReq)
		if err := req.validate(); err != nil {
			return addPolicyRes{}, err
		}

		err := svc.AddPolicy(ctx, auth.PolicyReq{
			Namespace:   req.Namespace,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object})
		if err != nil {
			return addPolicyRes{}, err
		}
		return addPolicyRes{authorized: true}, err
	}
}

func deletePolicyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(policyReq)
		if err := req.validate(); err != nil {
			return deletePolicyRes{}, err
		}

		err := svc.DeletePolicy(ctx, auth.PolicyReq{
			Namespace:   req.Namespace,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return deletePolicyRes{}, err
		}
		return deletePolicyRes{deleted: true}, nil
	}
}

func listObjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listObjectsReq)

		page, err := svc.ListObjects(ctx, auth.PolicyReq{
			Namespace:   req.Namespace,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		}, req.NextPageToken, req.Limit)
		if err != nil {
			return listObjectsRes{}, err
		}
		return listObjectsRes{policies: page.Policies, nextPageToken: page.NextPageToken}, nil
	}
}

func listAllObjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listObjectsReq)

		page, err := svc.ListAllObjects(ctx, auth.PolicyReq{
			Namespace:   req.Namespace,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return listObjectsRes{}, err
		}
		return listObjectsRes{policies: page.Policies, nextPageToken: page.NextPageToken}, nil
	}
}

func countObjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(countObjectsReq)

		count, err := svc.CountObjects(ctx, auth.PolicyReq{
			Namespace:   req.Namespace,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return countObjectsRes{}, err
		}
		return countObjectsRes{count: count}, nil
	}
}

func listSubjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listSubjectsReq)

		page, err := svc.ListSubjects(ctx, auth.PolicyReq{
			Namespace:   req.Namespace,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		}, req.NextPageToken, req.Limit)
		if err != nil {
			return listSubjectsRes{}, err
		}
		return listSubjectsRes{policies: page.Policies, nextPageToken: page.NextPageToken}, nil
	}
}

func listAllSubjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listSubjectsReq)

		page, err := svc.ListAllSubjects(ctx, auth.PolicyReq{
			Namespace:   req.Namespace,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return listSubjectsRes{}, err
		}
		return listSubjectsRes{policies: page.Policies, nextPageToken: page.NextPageToken}, nil
	}
}

func countSubjectsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(countSubjectsReq)

		count, err := svc.CountSubjects(ctx, auth.PolicyReq{
			Namespace:   req.Namespace,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return countSubjectsRes{}, err
		}
		return countSubjectsRes{count: count}, nil
	}
}

func assignEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignReq)

		if err := req.validate(); err != nil {
			return emptyRes{}, err
		}

		_, err := svc.Identify(ctx, req.token)
		if err != nil {
			return emptyRes{}, err
		}

		err = svc.Assign(ctx, req.token, req.memberID, req.groupID, req.groupType)
		if err != nil {
			return emptyRes{}, err
		}
		return emptyRes{}, nil

	}
}

func membersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(membersReq)
		if err := req.validate(); err != nil {
			return membersRes{}, err
		}

		pm := auth.PageMetadata{
			Offset: req.offset,
			Limit:  req.limit,
		}
		mp, err := svc.ListMembers(ctx, req.token, req.groupID, req.memberType, pm)
		if err != nil {
			return membersRes{}, err
		}
		var members []string
		for _, m := range mp.Members {
			members = append(members, m.ID)
		}
		return membersRes{
			offset:  req.offset,
			limit:   req.limit,
			total:   mp.PageMetadata.Total,
			members: members,
		}, nil
	}
}
