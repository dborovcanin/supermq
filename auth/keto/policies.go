// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package keto

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
	acl "github.com/ory/keto/proto/ory/keto/relation_tuples/v1alpha2"
)

const (
	subjectSetRegex = "^.{1,}:.{1,}#.{1,}$" // expected subject set structure is <namespace>:<object>#<relation>
	ketoNamespace   = "members"
)

type policyAgent struct {
	writer  acl.WriteServiceClient
	checker acl.CheckServiceClient
	reader  acl.ReadServiceClient
}

// NewPolicyAgent returns a gRPC communication functionalities
// to communicate with ORY Keto.
func NewPolicyAgent(checker acl.CheckServiceClient, writer acl.WriteServiceClient, reader acl.ReadServiceClient) auth.PolicyAgent {
	return policyAgent{checker: checker, writer: writer, reader: reader}
}

func (pa policyAgent) CheckPolicy(ctx context.Context, pr auth.PolicyReq) error {
	res, err := pa.checker.Check(context.Background(), &acl.CheckRequest{
		Namespace: pr.Namespace,
		Object:    pr.Object,
		Relation:  pr.Relation,
		Subject:   getSubject(pr),
	})
	if err != nil {
		return errors.Wrap(err, errors.ErrAuthorization)
	}
	if !res.GetAllowed() {
		return errors.ErrAuthorization
	}
	return nil
}

func (pa policyAgent) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {

	err := pa.CheckPolicy(ctx, pr)
	switch err {
	case errors.ErrAuthorization:
		var ss *acl.Subject
		switch isSubjectSet(pr.Subject) {
		case true:
			namespace, object, relation := parseSubjectSet(pr.Subject)
			ss = &acl.Subject{
				Ref: &acl.Subject_Set{Set: &acl.SubjectSet{Namespace: namespace, Object: object, Relation: relation}},
			}
		default:
			ss = &acl.Subject{Ref: &acl.Subject_Id{Id: pr.Subject}}
		}

		trt := pa.writer.TransactRelationTuples
		_, err := trt(context.Background(), &acl.TransactRelationTuplesRequest{
			RelationTupleDeltas: []*acl.RelationTupleDelta{
				{
					Action: acl.RelationTupleDelta_ACTION_INSERT,
					RelationTuple: &acl.RelationTuple{
						Namespace: pr.Namespace,
						Object:    pr.Object,
						Relation:  pr.Relation,
						Subject:   ss,
					},
				},
			},
		})
		return err
	case nil:
		return nil
	default:
		return err
	}
}

// AddPolicies
func (pa policyAgent) AddPolicies(ctx context.Context, pr []auth.PolicyReq) error {
	return fmt.Errorf("Not Implemented")
}

// DeletePolicies
func (pa policyAgent) DeletePolicies(ctx context.Context, pr []auth.PolicyReq) error {
	return fmt.Errorf("Not Implemented")
}

func (pa policyAgent) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	trt := pa.writer.TransactRelationTuples
	_, err := trt(context.Background(), &acl.TransactRelationTuplesRequest{
		RelationTupleDeltas: []*acl.RelationTupleDelta{
			{
				Action: acl.RelationTupleDelta_ACTION_DELETE,
				RelationTuple: &acl.RelationTuple{
					Namespace: pr.Namespace,
					Object:    pr.Object,
					Relation:  pr.Relation,
					Subject: &acl.Subject{Ref: &acl.Subject_Id{
						Id: pr.Subject,
					}},
				},
			},
		},
	})
	return err
}

func (pa policyAgent) RetrieveObjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) ([]auth.PolicyRes, string, error) {
	var ss *acl.Subject
	switch isSubjectSet(pr.Subject) {
	case true:
		namespace, object, relation := parseSubjectSet(pr.Subject)
		ss = &acl.Subject{
			Ref: &acl.Subject_Set{Set: &acl.SubjectSet{Namespace: namespace, Object: object, Relation: relation}},
		}
	default:
		ss = &acl.Subject{Ref: &acl.Subject_Id{Id: pr.Subject}}
	}

	query := &acl.ListRelationTuplesRequest_Query{
		Namespace: pr.Namespace,
		Relation:  pr.Relation,
		Subject:   ss,
	}

	res, err := pa.reader.ListRelationTuples(context.Background(), &acl.ListRelationTuplesRequest{
		Query:     query,
		PageToken: nextPageToken,
		PageSize:  limit,
	})
	return toPolicyRes(res.GetRelationTuples()), res.GetNextPageToken(), err

}

func (pa policyAgent) RetrieveAllObjects(ctx context.Context, pr auth.PolicyReq) ([]auth.PolicyRes, error) {
	var tuples []auth.PolicyRes
	nextPageToken := ""
	for {
		relationTuples, npt, err := pa.RetrieveObjects(ctx, pr, nextPageToken, 1000)
		if err != nil {
			return tuples, err
		}
		tuples = append(tuples, relationTuples...)
		if npt == "" {
			break
		}
		nextPageToken = npt
	}
	return tuples, nil
}

func (pa policyAgent) RetrieveAllObjectsCount(ctx context.Context, pr auth.PolicyReq) (int, error) {
	var count int
	nextPageToken := ""
	for {
		relationTuples, npt, err := pa.RetrieveObjects(ctx, pr, nextPageToken, 1000)
		if err != nil {
			return count, err
		}
		count = count + len(relationTuples)
		if npt == "" {
			break
		}
		nextPageToken = npt
	}
	return count, nil
}

func (pa policyAgent) RetrieveSubjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) ([]auth.PolicyRes, string, error) {
	query := &acl.ListRelationTuplesRequest_Query{
		Namespace: pr.Namespace,
		Relation:  pr.Relation,
		Object:    pr.Object,
	}

	res, err := pa.reader.ListRelationTuples(context.Background(), &acl.ListRelationTuplesRequest{
		Query:     query,
		PageToken: nextPageToken,
		PageSize:  limit,
	})
	return toPolicyRes(res.GetRelationTuples()), res.GetNextPageToken(), err
}

func (pa policyAgent) RetrieveAllSubjects(ctx context.Context, pr auth.PolicyReq) ([]auth.PolicyRes, error) {
	var tuples []auth.PolicyRes
	nextPageToken := ""
	for {
		relationTuples, npt, err := pa.RetrieveSubjects(ctx, pr, nextPageToken, 1000)
		if err != nil {
			return tuples, err
		}
		tuples = append(tuples, relationTuples...)
		if npt == "" {
			break
		}
		nextPageToken = npt
	}
	return tuples, nil
}

func (pa policyAgent) RetrieveAllSubjectsCount(ctx context.Context, pr auth.PolicyReq) (int, error) {
	var count int
	nextPageToken := ""
	for {
		relationTuples, npt, err := pa.RetrieveSubjects(ctx, pr, nextPageToken, 1000)
		if err != nil {
			return count, err
		}
		count = count + len(relationTuples)
		if npt == "" {
			break
		}
		nextPageToken = npt
	}
	return count, nil
}

// getSubject returns a 'subject' field for ACL(access control lists).
// If the given PolicyReq argument contains a subject as subject set,
// it returns subject set; otherwise, it returns a subject.
func getSubject(pr auth.PolicyReq) *acl.Subject {
	if isSubjectSet(pr.Subject) {
		namespace, object, relation := parseSubjectSet(pr.Subject)
		return &acl.Subject{
			Ref: &acl.Subject_Set{Set: &acl.SubjectSet{
				Namespace: namespace,
				Object:    object,
				Relation:  relation,
			}},
		}
	}

	return &acl.Subject{Ref: &acl.Subject_Id{Id: pr.Subject}}
}

// isSubjectSet returns true when given subject is subject set.
// Otherwise, it returns false.
func isSubjectSet(subject string) bool {
	r, err := regexp.Compile(subjectSetRegex)
	if err != nil {
		return false
	}
	return r.MatchString(subject)
}

func parseSubjectSet(subjectSet string) (namespace, object, relation string) {
	r := strings.Split(subjectSet, ":")
	if len(r) != 2 {
		return
	}
	namespace = r[0]

	r = strings.Split(r[1], "#")
	if len(r) != 2 {
		return
	}

	object = r[0]
	relation = r[1]

	return
}

func toPolicyRes(rts []*acl.RelationTuple) []auth.PolicyRes {
	policies := make([]auth.PolicyRes, len(rts))
	for _, rt := range rts {
		policies = append(policies, auth.PolicyRes{
			Namespace: rt.Namespace,
			Object:    rt.Object,
			Relation:  rt.Relation,
			Subject:   rt.Subject.String(),
		})
	}
	return policies
}
