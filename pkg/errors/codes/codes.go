// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package codes defines error codes used throughout SuperMQ.
// Each code carries metadata about how it should be presented to API clients.
package codes

import "net/http"

// Code represents a typed error code with associated metadata.
// The metadata defines how errors with this code should be presented
// to API clients, including the HTTP status code and which context
// keys are safe to expose in the response.
type Code struct {
	id         string
	message    string   // User-facing message (safe to expose)
	statusCode int      // HTTP status code
	expose     []string // Context keys safe to expose in API response
}

// ID returns the unique identifier for this error code.
func (c Code) ID() string { return c.id }

// Message returns the user-facing message for this error code.
func (c Code) Message() string { return c.message }

// StatusCode returns the HTTP status code for this error code.
func (c Code) StatusCode() int { return c.statusCode }

// Expose returns the list of context keys that are safe to expose
// in API responses for errors with this code.
func (c Code) Expose() []string { return c.expose }

// IsZero returns true if this is an uninitialized/empty code.
func (c Code) IsZero() bool { return c.id == "" }

// NewCode creates a new error code with the given parameters.
func NewCode(id, message string, statusCode int, expose ...string) Code {
	return Code{
		id:         id,
		message:    message,
		statusCode: statusCode,
		expose:     expose,
	}
}

// Predefined error codes for common error scenarios.
// These codes are used throughout the codebase to ensure consistent
// error handling and API responses.
var (
	// Entity operation errors (422 Unprocessable Entity)
	CreateFailed = NewCode(
		"CREATE_FAILED",
		"Failed to create the resource",
		http.StatusUnprocessableEntity,
		"entity_type", "operation",
	)

	UpdateFailed = NewCode(
		"UPDATE_FAILED",
		"Failed to update the resource",
		http.StatusUnprocessableEntity,
		"entity_type", "operation",
	)

	DeleteFailed = NewCode(
		"DELETE_FAILED",
		"Failed to delete the resource",
		http.StatusUnprocessableEntity,
		"entity_type",
	)

	ViewFailed = NewCode(
		"VIEW_FAILED",
		"Failed to retrieve the resource",
		http.StatusUnprocessableEntity,
		"entity_type",
	)

	// State errors
	NotFound = NewCode(
		"NOT_FOUND",
		"Resource not found",
		http.StatusNotFound,
		"entity_type", "entity_id",
	)

	Conflict = NewCode(
		"CONFLICT",
		"Resource already exists",
		http.StatusConflict,
		"entity_type", "field",
	)

	InvalidStatus = NewCode(
		"INVALID_STATUS",
		"Invalid status value",
		http.StatusBadRequest,
		"entity_type", "field", "reason",
	)

	// Authentication errors (401 Unauthorized)
	Unauthenticated = NewCode(
		"UNAUTHENTICATED",
		"Authentication required",
		http.StatusUnauthorized,
	)

	InvalidCredentials = NewCode(
		"INVALID_CREDENTIALS",
		"Invalid credentials provided",
		http.StatusUnauthorized,
	)

	// Authorization errors (403 Forbidden)
	Unauthorized = NewCode(
		"UNAUTHORIZED",
		"You don't have permission to perform this action",
		http.StatusForbidden,
		"required_permission",
	)

	// Validation errors (400 Bad Request)
	ValidationFailed = NewCode(
		"VALIDATION_FAILED",
		"Invalid request",
		http.StatusBadRequest,
		"field", "reason",
	)

	MalformedEntity = NewCode(
		"MALFORMED_ENTITY",
		"Invalid request data",
		http.StatusBadRequest,
		"field", "reason",
	)

	// Domain-specific codes - Clients
	ClientEnableFailed = NewCode(
		"CLIENT_ENABLE_FAILED",
		"Failed to enable client",
		http.StatusUnprocessableEntity,
		"entity_type", "entity_id", "reason",
	)

	ClientDisableFailed = NewCode(
		"CLIENT_DISABLE_FAILED",
		"Failed to disable client",
		http.StatusUnprocessableEntity,
		"entity_type", "entity_id", "reason",
	)

	// Domain-specific codes - Channels
	ChannelEnableFailed = NewCode(
		"CHANNEL_ENABLE_FAILED",
		"Failed to enable channel",
		http.StatusUnprocessableEntity,
		"entity_type", "entity_id", "reason",
	)

	ChannelDisableFailed = NewCode(
		"CHANNEL_DISABLE_FAILED",
		"Failed to disable channel",
		http.StatusUnprocessableEntity,
		"entity_type", "entity_id", "reason",
	)

	// Domain-specific codes - Groups
	GroupEnableFailed = NewCode(
		"GROUP_ENABLE_FAILED",
		"Failed to enable group",
		http.StatusUnprocessableEntity,
		"entity_type", "entity_id", "reason",
	)

	GroupDisableFailed = NewCode(
		"GROUP_DISABLE_FAILED",
		"Failed to disable group",
		http.StatusUnprocessableEntity,
		"entity_type", "entity_id", "reason",
	)

	// Domain-specific codes - Users
	UserEnableFailed = NewCode(
		"USER_ENABLE_FAILED",
		"Failed to enable user",
		http.StatusUnprocessableEntity,
		"entity_type", "entity_id", "reason",
	)

	UserDisableFailed = NewCode(
		"USER_DISABLE_FAILED",
		"Failed to disable user",
		http.StatusUnprocessableEntity,
		"entity_type", "entity_id", "reason",
	)

	// Policy errors
	AddPoliciesFailed = NewCode(
		"ADD_POLICIES_FAILED",
		"Failed to add policies",
		http.StatusUnprocessableEntity,
		"entity_type", "operation",
	)

	DeletePoliciesFailed = NewCode(
		"DELETE_POLICIES_FAILED",
		"Failed to remove policies",
		http.StatusUnprocessableEntity,
		"entity_type", "operation",
	)

	InvalidPolicy = NewCode(
		"INVALID_POLICY",
		"Invalid policy",
		http.StatusBadRequest,
		"field", "reason",
	)

	// Invitation errors
	InvitationAlreadyAccepted = NewCode(
		"INVITATION_ALREADY_ACCEPTED",
		"Invitation has already been accepted",
		http.StatusConflict,
	)

	InvitationAlreadyRejected = NewCode(
		"INVITATION_ALREADY_REJECTED",
		"Invitation has already been rejected",
		http.StatusConflict,
	)

	// Database operation errors (internal, rarely exposed directly)
	DBOperationFailed = NewCode(
		"DB_OPERATION_FAILED",
		"Database operation failed",
		http.StatusInternalServerError,
	)

	// Rollback errors
	RollbackFailed = NewCode(
		"ROLLBACK_FAILED",
		"Failed to rollback operation",
		http.StatusInternalServerError,
	)

	// Generic internal error (fallback)
	InternalError = NewCode(
		"INTERNAL_ERROR",
		"An unexpected error occurred",
		http.StatusInternalServerError,
	)
)
