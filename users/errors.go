// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package users

import (
	"github.com/absmach/magistrala/pkg/errors"
	autherr "github.com/absmach/magistrala/pkg/errors/auth"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

const (
	// Malformed errors
	HashErr   = "failed to hash user credentials"
	StatusErr = "invalid status"
	RoleErr   = "invalid role"

	// AuthN and AuthZ errors
	AuthNErr               = "failed to authenticate user"
	AuthZErr               = "failed to authorize user"
	InvalidPasswordErr     = "invalid password"
	ResetTokenErr          = "failed to issue reset token"
	DisabledUserRefreshErr = "failed to refresh token for disabled user"

	// Internal errors
	ViewErr            = "failed to fetch user"
	IssueTokenErr      = "failed to issue token"
	RollbackErr        = "failed to remove policies during rollback"
	AddPoliciesErr     = "failed to add user policies"
	DeletePoliciesErr  = "failed to delete user policies"
	PermissionsListErr = "failed to list permissions"
	UpdateErr          = "failed to update user"
	UpdateTagsErr      = "failed to update user tags"
	UpdateIdentityErr  = "failed to update user identity"
	UpdateSecretErr    = "failed to update user secret"
	UpdateRoleErr      = "failed to update user role"
	ListMembersErr     = "failed to list members"
	UserAddErr         = "failed to add user"
	UpdateStatus       = "failed to update user status to "
)

var (
	errNotAddedStatus   = errors.New("response status is not added")
	errNotDeletedStatus = errors.New("response status is not deleted")
	errAuthZ            = autherr.NewAuthZError(AuthZErr, nil)
)

type (
	MalformedError = svcerr.MalformedError
	NotFoundError  = svcerr.NotFoundError
	InternalError  = svcerr.OtherError
)

func newMalformedError(text string, err error) error {
	return &MalformedError{CustomError: errors.NewErr(text, err)}
}

func newNotFoundError(text string, err error) error {
	return &NotFoundError{CustomError: errors.NewErr(text, err)}
}

func newInternalError(text string, err error) error {
	return &InternalError{CustomError: errors.NewErr(text, err)}
}
