// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/mainflux/mainflux/internal/apiutil"

// authReq represents authorization request. It contains:
// 1. subject - an action invoker
// 2. object - an entity over which action will be executed
// 3. action - type of action that will be executed (read/write)
type authReq struct {
	Namespace   string
	SubjectType string
	SubjectKind string
	Subject     string
	Relation    string
	Permission  string
	ObjectType  string
	Object      string
}

func (req authReq) validate() error {
	if req.Subject == "" {
		return apiutil.ErrMissingPolicySub
	}

	if req.Object == "" {
		return apiutil.ErrMissingPolicyObj
	}

	// if req.SubjectKind == "" {
	// 	return apiutil.ErrMissingPolicySub
	// }

	if req.Permission == "" {
		return apiutil.ErrMalformedPolicyAct
	}

	return nil
}
