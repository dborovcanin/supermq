// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
	acl "github.com/ory/keto/proto/ory/keto/relation_tuples/v1alpha2"
	"google.golang.org/grpc"
)

type MockSubjectSet struct {
	Object   string
	Relation string
}

type policyAgentMock struct {
	mu sync.Mutex
	// authzDb stores 'subject' as a key, and subject policies as a value.
	authzDB map[string][]MockSubjectSet
}

// NewKetoMock returns a mock service for Keto.
// This mock is not implemented yet.
func NewKetoMock(db map[string][]MockSubjectSet) auth.PolicyAgent {
	return &policyAgentMock{authzDB: db}
}

func (pa *policyAgentMock) CheckPolicy(ctx context.Context, pr auth.PolicyReq) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	ssList := pa.authzDB[pr.Subject]
	for _, ss := range ssList {
		if ss.Object == pr.Object && ss.Relation == pr.Relation {
			return nil
		}
	}
	return errors.ErrAuthorization
}

func (pa *policyAgentMock) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.authzDB[pr.Subject] = append(pa.authzDB[pr.Subject], MockSubjectSet{Object: pr.Object, Relation: pr.Relation})
	return nil
}

func (pa *policyAgentMock) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	ssList := pa.authzDB[pr.Subject]
	for k, ss := range ssList {
		if ss.Object == pr.Object && ss.Relation == pr.Relation {
			ssList[k] = MockSubjectSet{}
		}
	}
	return nil
}

func (pa *policyAgentMock) RetrieveObjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) ([]acl.RelationTuple, string, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	ssList := pa.authzDB[pr.Subject]
	tuple := []acl.RelationTuple{}
	for _, ss := range ssList {
		if ss.Relation == pr.Relation {
			tuple = append(tuple, acl.RelationTuple{Object: ss.Object, Relation: ss.Relation})
		}
	}
	return tuple, "", nil
}

// RetrieveAllPolicies
func (pa *policyAgentMock) RetrieveAllObjects(ctx context.Context, pr auth.PolicyReq) ([]acl.RelationTuple, error) {
	return nil, nil
}

func (pa *policyAgentMock) RetrieveAllObjectsCount(ctx context.Context, pr auth.PolicyReq) (int, error) {
	return 0, nil
}

// RetrieveAllPolicies
func (pa *policyAgentMock) RetrieveAllSubjects(ctx context.Context, pr auth.PolicyReq) ([]acl.RelationTuple, error) {
	return nil, nil
}
func (pa *policyAgentMock) RetrieveAllSubjectsCount(ctx context.Context, pr auth.PolicyReq) (int, error) {
	return 0, nil
}

// DeletePolicies
func (pa *policyAgentMock) DeletePolicies(ctx context.Context, pr []auth.PolicyReq) error {
	return nil
}

// AddPolicies
func (pa *policyAgentMock) AddPolicies(ctx context.Context, pr []auth.PolicyReq) error {
	return nil
}

// RetrieveAllPolicies
func (pa *policyAgentMock) RetrieveSubjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) ([]acl.RelationTuple, string, error) {
	return nil, "", nil
}

func (svc *policyAgentMock) ListAllObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	panic("not implemented")
}

func (svc *policyAgentMock) CountObjects(ctx context.Context, req *mainflux.CountObjectsReq, _ ...grpc.CallOption) (r *mainflux.CountObjectsRes, err error) {
	panic("not implemented")
}

func (svc *policyAgentMock) ListSubjects(ctx context.Context, req *mainflux.ListSubjectsReq, _ ...grpc.CallOption) (r *mainflux.ListSubjectsRes, err error) {
	panic("not implemented")
}
func (svc *policyAgentMock) ListAllSubjects(ctx context.Context, req *mainflux.ListSubjectsReq, _ ...grpc.CallOption) (r *mainflux.ListSubjectsRes, err error) {
	panic("not implemented")
}
func (svc *policyAgentMock) CountSubjects(ctx context.Context, req *mainflux.CountSubjectsReq, _ ...grpc.CallOption) (r *mainflux.CountSubjectsRes, err error) {
	panic("not implemented")
}
