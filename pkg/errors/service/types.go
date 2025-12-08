// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package service provides service-layer error codes.
// Import this package to access error codes appropriate for service layer operations.
package service

import (
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/errors/codes"
)

// Service layer error codes.
// Use with errors.E(): errors.E(svcerr.NotFound).Wrap(err)
var (
	// Authentication errors
	Unauthenticated    = codes.Unauthenticated
	InvalidCredentials = codes.InvalidCredentials

	// Authorization errors
	Unauthorized = codes.Unauthorized

	// Validation errors
	MalformedEntity  = codes.MalformedEntity
	ValidationFailed = codes.ValidationFailed
	InvalidStatus    = codes.InvalidStatus
	InvalidPolicy    = codes.InvalidPolicy

	// Entity state errors
	NotFound = codes.NotFound
	Conflict = codes.Conflict

	// Entity operation errors
	CreateFailed = codes.CreateFailed
	ViewFailed   = codes.ViewFailed
	UpdateFailed = codes.UpdateFailed
	DeleteFailed = codes.DeleteFailed

	// Client/User state errors
	ClientEnableFailed  = codes.ClientEnableFailed
	ClientDisableFailed = codes.ClientDisableFailed
	UserEnableFailed    = codes.UserEnableFailed
	UserDisableFailed   = codes.UserDisableFailed

	// Policy errors
	AddPoliciesFailed    = codes.AddPoliciesFailed
	DeletePoliciesFailed = codes.DeletePoliciesFailed

	// Invitation errors
	InvitationAlreadyAccepted = codes.InvitationAlreadyAccepted
	InvitationAlreadyRejected = codes.InvitationAlreadyRejected

	// Rollback errors
	RollbackFailed = codes.RollbackFailed
)

// Legacy error values for backward compatibility.
var (
	ErrAuthentication                     = errors.New("failed to perform authentication over the entity")
	ErrAuthorization                      = errors.New("failed to perform authorization over the entity")
	ErrDomainAuthorization                = errors.New("failed to perform authorization over the domain")
	ErrLogin                              = errors.New("invalid credentials")
	ErrMalformedEntity                    = errors.New("malformed entity specification")
	ErrNotFound                           = errors.New("entity not found")
	ErrConflict                           = errors.New("entity already exists")
	ErrCreateEntity                       = errors.New("failed to create entity")
	ErrRemoveEntity                       = errors.New("failed to remove entity")
	ErrViewEntity                         = errors.New("view entity failed")
	ErrUpdateEntity                       = errors.New("update entity failed")
	ErrInvalidStatus                      = errors.New("invalid status")
	ErrInvalidRole                        = errors.New("invalid client role")
	ErrInvalidPolicy                      = errors.New("invalid policy")
	ErrEnableClient                       = errors.New("failed to enable client")
	ErrDisableClient                      = errors.New("failed to disable client")
	ErrAddPolicies                        = errors.New("failed to add policies")
	ErrDeletePolicies                     = errors.New("failed to remove policies")
	ErrSearch                             = errors.New("failed to search clients")
	ErrInvitationAlreadyRejected          = errors.New("invitation already rejected")
	ErrInvitationAlreadyAccepted          = errors.New("invitation already accepted")
	ErrParentGroupAuthorization           = errors.New("failed to authorize parent group")
	ErrMissingUsername                    = errors.New("missing usernames")
	ErrEnableUser                         = errors.New("failed to enable user")
	ErrDisableUser                        = errors.New("failed to disable user")
	ErrRollbackRepo                       = errors.New("failed to rollback repo")
	ErrUnauthorizedPAT                    = errors.New("failed to authorize PAT")
	ErrRetainOneMember                    = errors.New("must retain at least one member")
	ErrSuperAdminAction                   = errors.New("not authorized to perform admin action")
	ErrUserAlreadyVerified                = errors.New("user already verified")
	ErrInvalidUserVerification            = errors.New("invalid verification")
	ErrUserVerificationExpired            = errors.New("verification expired, please generate new verification")
	ErrExternalAuthProviderCouldNotUpdate = errors.New("account details can only be updated through your authentication provider's settings")
)
