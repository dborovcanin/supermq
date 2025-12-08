// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package repository contains repository-layer error definitions.
package repository

import (
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/errors/codes"
)

// Repository layer errors.
// These errors are returned by repository layer operations and carry
// error codes for consistent API responses. Note that the codes are
// the same as service layer codes - the code identifies the type of
// failure, not the layer it occurred in.
var (
	// Entity validation errors
	ErrMalformedEntity = errors.NewWithCode(codes.MalformedEntity, "malformed entity specification")

	// Entity state errors
	ErrNotFound = errors.NewWithCode(codes.NotFound, "entity not found")
	ErrConflict = errors.NewWithCode(codes.Conflict, "entity already exists")

	// Entity operation errors
	ErrCreateEntity = errors.NewWithCode(codes.CreateFailed, "failed to create entity in the db")
	ErrViewEntity   = errors.NewWithCode(codes.ViewFailed, "failed to retrieve entity from db")
	ErrUpdateEntity = errors.NewWithCode(codes.UpdateFailed, "failed to update entity in db")
	ErrRemoveEntity = errors.NewWithCode(codes.DeleteFailed, "failed to remove entity from db")

	// Database operation errors
	ErrFailedOpDB = errors.NewWithCode(codes.DBOperationFailed, "operation on db element failed")

	// Group retrieval errors
	ErrFailedToRetrieveAllGroups = errors.NewWithCode(codes.ViewFailed, "failed to retrieve all groups")

	// Migration errors
	ErrRoleMigration = errors.NewWithCode(codes.InternalError, "failed to apply role migration")

	// Validation errors
	ErrMissingNames = errors.NewWithCode(codes.ValidationFailed, "missing first or last name")
)
