// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package service contains service-layer error definitions.
package service

import (
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/errors/codes"
)

// Service layer errors.
// These errors are returned by service layer operations and carry
// error codes for consistent API responses.
var (
	// Authentication errors
	ErrAuthentication = errors.NewWithCode(codes.Unauthenticated, "failed to perform authentication over the entity")
	ErrLogin          = errors.NewWithCode(codes.InvalidCredentials, "invalid credentials")

	// Authorization errors
	ErrAuthorization       = errors.NewWithCode(codes.Unauthorized, "failed to perform authorization over the entity")
	ErrDomainAuthorization = errors.NewWithCode(codes.Unauthorized, "failed to perform authorization over the domain")
	ErrUnauthorizedPAT     = errors.NewWithCode(codes.Unauthorized, "failed to authorize PAT")
	ErrSuperAdminAction    = errors.NewWithCode(codes.Unauthorized, "not authorized to perform admin action")

	// Entity validation errors
	ErrMalformedEntity = errors.NewWithCode(codes.MalformedEntity, "malformed entity specification")
	ErrInvalidStatus   = errors.NewWithCode(codes.InvalidStatus, "invalid status")
	ErrInvalidRole     = errors.NewWithCode(codes.ValidationFailed, "invalid client role")
	ErrInvalidPolicy   = errors.NewWithCode(codes.InvalidPolicy, "invalid policy")

	// Entity state errors
	ErrNotFound = errors.NewWithCode(codes.NotFound, "entity not found")
	ErrConflict = errors.NewWithCode(codes.Conflict, "entity already exists")

	// Entity operation errors
	ErrCreateEntity = errors.NewWithCode(codes.CreateFailed, "failed to create entity")
	ErrViewEntity   = errors.NewWithCode(codes.ViewFailed, "failed to retrieve entity")
	ErrUpdateEntity = errors.NewWithCode(codes.UpdateFailed, "failed to update entity")
	ErrRemoveEntity = errors.NewWithCode(codes.DeleteFailed, "failed to remove entity")

	// Client-specific errors
	ErrEnableClient  = errors.NewWithCode(codes.ClientEnableFailed, "failed to enable client")
	ErrDisableClient = errors.NewWithCode(codes.ClientDisableFailed, "failed to disable client")

	// User-specific errors
	ErrEnableUser  = errors.NewWithCode(codes.UserEnableFailed, "failed to enable user")
	ErrDisableUser = errors.NewWithCode(codes.UserDisableFailed, "failed to disable user")

	// Policy errors
	ErrAddPolicies    = errors.NewWithCode(codes.AddPoliciesFailed, "failed to add policies")
	ErrDeletePolicies = errors.NewWithCode(codes.DeletePoliciesFailed, "failed to remove policies")

	// Search errors
	ErrSearch = errors.NewWithCode(codes.ViewFailed, "failed to search clients")

	// Invitation errors
	ErrInvitationAlreadyRejected = errors.NewWithCode(codes.InvitationAlreadyRejected, "invitation already rejected")
	ErrInvitationAlreadyAccepted = errors.NewWithCode(codes.InvitationAlreadyAccepted, "invitation already accepted")

	// Parent group errors
	ErrParentGroupAuthorization = errors.NewWithCode(codes.Unauthorized, "failed to authorize parent group")

	// User validation errors
	ErrMissingUsername = errors.NewWithCode(codes.ValidationFailed, "missing usernames")

	// Rollback errors
	ErrRollbackRepo = errors.NewWithCode(codes.RollbackFailed, "failed to rollback repo")

	// Member errors
	ErrRetainOneMember = errors.NewWithCode(codes.ValidationFailed, "must retain at least one member")

	// User verification errors
	ErrUserAlreadyVerified     = errors.NewWithCode(codes.Conflict, "user already verified")
	ErrInvalidUserVerification = errors.NewWithCode(codes.ValidationFailed, "invalid verification")
	ErrUserVerificationExpired = errors.NewWithCode(codes.ValidationFailed, "verification expired, please generate new verification")

	// External auth provider errors
	ErrExternalAuthProviderCouldNotUpdate = errors.NewWithCode(codes.ValidationFailed, "account details can only be updated through your authentication provider's settings")
)
