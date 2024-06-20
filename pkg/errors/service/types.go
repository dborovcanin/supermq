// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package service

import "github.com/absmach/magistrala/pkg/errors"

// Wrapper service type errors
type (

	// AuthenticationError indicates failure occurred while authenticating the entity.
	AuthenticationError struct {
		*errors.CustomError
	}

	// AuthorizationError indicates failure occurred while authorizing the entity.
	AuthorizationError struct {
		*errors.CustomError
	}

	// MalformedEntityError indicates a malformed entity specification.
	MalformedError struct {
		*errors.CustomError
	}

	// ConflictError indicates unique constraint violation.
	ConflictError struct {
		*errors.CustomError
	}

	// NotFoundError indicates that resource is not found at the given location.
	NotFoundError struct {
		*errors.CustomError
	}

	// OtherError indicates unknown error usually caused by internal.
	OtherError struct {
		*errors.CustomError
	}
)

func NewAuthNError(err error) error {
	return &AuthenticationError{errors.NewErr("failed to perform authentication over the user", err)}
}

func NewAuthZError(err error) error {
	return &AuthenticationError{errors.NewErr("failed to perform authorization over the user", err)}
}
