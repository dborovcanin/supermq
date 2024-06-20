// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package service

import "github.com/absmach/magistrala/pkg/errors"

// Common errors that can be found in service layer.
var (
	// ErrAuthentication indicates failure occurred while authenticating the entity.
	ErrAuthentication = &AuthenticationError{errors.NewErr("failed to perform authentication over the entity", nil)}

	// ErrAuthorization indicates failure occurred while authorizing the entity.
	ErrAuthorization = &AuthorizationError{errors.NewErr("failed to perform authorization over the entity", nil)}

	// ErrDomainAuthorization indicates failure occurred while authorizing the domain.
	ErrDomainAuthorization = &AuthorizationError{errors.NewErr("failed to perform authorization over the domain", nil)}

	// ErrLogin indicates wrong login credentials.
	ErrLogin = &AuthenticationError{errors.NewErr("invalid user id or secret", nil)}

	// ErrMalformedEntity indicates a malformed entity specification.
	ErrMalformedEntity = &MalformedError{errors.NewErr("malformed entity specification", nil)}

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = &NotFoundError{errors.NewErr("entity not found", nil)}

	// ErrConflict indicates that entity already exists.
	ErrConflict = &ConflictError{errors.NewErr("entity already exists", nil)}

	// ErrCreateEntity indicates error in creating entity or entities.
	ErrCreateEntity = &OtherError{errors.NewErr("failed to create entity", nil)}

	// ErrRemoveEntity indicates error in removing entity.
	ErrRemoveEntity = &OtherError{errors.NewErr("failed to remove entity", nil)}

	// ErrViewEntity indicates error in viewing entity or entities.
	ErrViewEntity = &OtherError{errors.NewErr("view entity failed", nil)}

	// ErrUpdateEntity indicates error in updating entity or entities.
	ErrUpdateEntity = &OtherError{errors.NewErr("update entity failed", nil)}

	// ErrInvalidStatus indicates an invalid status.
	ErrInvalidStatus = &OtherError{errors.NewErr("invalid status", nil)}

	// ErrInvalidRole indicates that an invalid role.
	ErrInvalidRole = &OtherError{errors.NewErr("invalid client role", nil)}

	// ErrInvalidPolicy indicates that an invalid policy.
	ErrInvalidPolicy = &OtherError{errors.NewErr("invalid policy", nil)}

	// ErrEnableClient indicates error in enabling client.
	ErrEnableClient = &OtherError{errors.NewErr("failed to enable client", nil)}

	// ErrDisableClient indicates error in disabling client.
	ErrDisableClient = &OtherError{errors.NewErr("failed to disable client", nil)}

	// ErrAddPolicies indicates error in adding policies.
	ErrAddPolicies = &OtherError{errors.NewErr("failed to add policies", nil)}

	// ErrDeletePolicies indicates error in removing policies.
	ErrDeletePolicies = &OtherError{errors.NewErr("failed to remove policies", nil)}
)
