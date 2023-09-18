// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

type identityRes struct {
	id    string
	email string
}

type issueRes struct {
	value string
	extra map[string]interface{}
}

type authorizeRes struct {
	authorized bool
}

type addPolicyRes struct {
	authorized bool
}

type deletePolicyRes struct {
	deleted bool
}

type listObjectsRes struct {
	policies      []string
	nextPageToken string
}

type countObjectsRes struct {
	count int
}

type listSubjectsRes struct {
	policies      []string
	nextPageToken string
}

type countSubjectsRes struct {
	count int
}

type membersRes struct {
	total     uint64
	offset    uint64
	limit     uint64
	groupType string
	members   []string
}
type emptyRes struct {
	err error
}
