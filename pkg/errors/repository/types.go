// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package repository provides repository-layer error codes.
// Import this package to access error codes appropriate for repository layer operations.
package repository

import (
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/errors/codes"
)

// Repository layer error codes.
// Use with errors.E(): errors.E(repoerr.NotFound).Wrap(err)
var (
	// Validation errors
	MalformedEntity = codes.MalformedEntity

	// Entity state errors
	NotFound = codes.NotFound
	Conflict = codes.Conflict

	// Entity operation errors
	CreateFailed = codes.CreateFailed
	ViewFailed   = codes.ViewFailed
	UpdateFailed = codes.UpdateFailed
	DeleteFailed = codes.DeleteFailed

	// Database operation errors
	DBOperationFailed = codes.DBOperationFailed
)

// Legacy error values for backward compatibility.
var (
	ErrMalformedEntity           = errors.New("malformed entity specification")
	ErrNotFound                  = errors.New("entity not found")
	ErrConflict                  = errors.New("entity already exists")
	ErrCreateEntity              = errors.New("failed to create entity in the db")
	ErrViewEntity                = errors.New("view entity failed")
	ErrUpdateEntity              = errors.New("update entity failed")
	ErrRemoveEntity              = errors.New("failed to remove entity")
	ErrFailedOpDB                = errors.New("operation on db element failed")
	ErrFailedToRetrieveAllGroups = errors.New("failed to retrieve all groups")
	ErrRoleMigration             = errors.New("failed to apply role migration")
	ErrMissingNames              = errors.New("missing first or last name")
)
