// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package repository

import "github.com/absmach/magistrala/pkg/errors"

// Wrapper for Repository errors.
var (
	// ErrMalformedEntity indicates a malformed entity specification.
	ErrMalformedEntity = &ConstraintError{errors.NewErr("malformed entity specification", nil)}

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = &ReadError{errors.NewErr("entity not found", nil)}

	// ErrConflict indicates that entity already exists.
	ErrConflict = &WriteError{errors.NewErr("entity already exists", nil)}

	// ErrCreateEntity indicates error in creating entity or entities.
	ErrCreateEntity = &WriteError{errors.NewErr("failed to create entity in the db", nil)}

	// ErrViewEntity indicates error in viewing entity or entities.
	ErrViewEntity = &ReadError{errors.NewErr("view entity failed", nil)}

	// ErrUpdateEntity indicates error in updating entity or entities.
	ErrUpdateEntity = &WriteError{errors.NewErr("update entity failed", nil)}

	// ErrRemoveEntity indicates error in removing entity.
	ErrRemoveEntity = &WriteError{errors.NewErr("failed to remove entity", nil)}

	// ErrFailedOpDB indicates a failure in a database operation.
	ErrFailedOpDB = &ConstraintError{errors.NewErr("operation on db element failed", nil)}

	// ErrFailedToRetrieveAllGroups failed to retrieve groups.
	ErrFailedToRetrieveAllGroups = &ReadError{errors.NewErr("failed to retrieve all groups", nil)}
)
