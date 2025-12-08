// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package codes defines error codes used throughout SuperMQ.
// Each code carries metadata about how it should be presented to API clients.
package codes

import "net/http"

// Code represents a typed error code with associated metadata.
// Codes are immutable and should be defined as package-level variables.
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

// Expose returns the list of context keys that are safe to expose.
func (c Code) Expose() []string { return c.expose }

// IsZero returns true if this is an uninitialized/empty code.
func (c Code) IsZero() bool { return c.id == "" }

// New creates a new error code.
func New(id, message string, statusCode int, expose ...string) Code {
	return Code{
		id:         id,
		message:    message,
		statusCode: statusCode,
		expose:     expose,
	}
}

// Predefined error codes.
var (
	// Entity operation errors
	CreateFailed = New("CREATE_FAILED", "Failed to create the resource", http.StatusUnprocessableEntity, "entity_type")
	ViewFailed   = New("VIEW_FAILED", "Failed to retrieve the resource", http.StatusUnprocessableEntity, "entity_type")
	UpdateFailed = New("UPDATE_FAILED", "Failed to update the resource", http.StatusUnprocessableEntity, "entity_type")
	DeleteFailed = New("DELETE_FAILED", "Failed to delete the resource", http.StatusUnprocessableEntity, "entity_type")

	// Entity state errors
	NotFound = New("NOT_FOUND", "Resource not found", http.StatusNotFound, "entity_type", "entity_id")
	Conflict = New("CONFLICT", "Resource already exists", http.StatusConflict, "entity_type", "entity_id")

	// Validation errors
	MalformedEntity  = New("MALFORMED_ENTITY", "Request body contains invalid data", http.StatusBadRequest, "entity_type", "field")
	ValidationFailed = New("VALIDATION_FAILED", "Validation failed", http.StatusBadRequest, "field", "reason")
	InvalidStatus    = New("INVALID_STATUS", "Invalid status value", http.StatusBadRequest, "status")
	InvalidPolicy    = New("INVALID_POLICY", "Invalid policy", http.StatusBadRequest)

	// Authentication errors
	Unauthenticated    = New("UNAUTHENTICATED", "Authentication required", http.StatusUnauthorized)
	InvalidCredentials = New("INVALID_CREDENTIALS", "Invalid credentials", http.StatusUnauthorized)

	// Authorization errors
	Unauthorized = New("UNAUTHORIZED", "Access denied", http.StatusForbidden, "entity_type", "entity_id", "permission")

	// Client/User state errors
	ClientEnableFailed  = New("CLIENT_ENABLE_FAILED", "Failed to enable client", http.StatusUnprocessableEntity)
	ClientDisableFailed = New("CLIENT_DISABLE_FAILED", "Failed to disable client", http.StatusUnprocessableEntity)
	UserEnableFailed    = New("USER_ENABLE_FAILED", "Failed to enable user", http.StatusUnprocessableEntity)
	UserDisableFailed   = New("USER_DISABLE_FAILED", "Failed to disable user", http.StatusUnprocessableEntity)

	// Policy errors
	AddPoliciesFailed    = New("ADD_POLICIES_FAILED", "Failed to add policies", http.StatusUnprocessableEntity)
	DeletePoliciesFailed = New("DELETE_POLICIES_FAILED", "Failed to delete policies", http.StatusUnprocessableEntity)

	// Invitation errors
	InvitationAlreadyAccepted = New("INVITATION_ALREADY_ACCEPTED", "Invitation was already accepted", http.StatusConflict)
	InvitationAlreadyRejected = New("INVITATION_ALREADY_REJECTED", "Invitation was already rejected", http.StatusConflict)

	// Database errors
	DBOperationFailed = New("DB_OPERATION_FAILED", "Database operation failed", http.StatusInternalServerError)

	// Rollback errors
	RollbackFailed = New("ROLLBACK_FAILED", "Rollback operation failed", http.StatusInternalServerError)

	// Internal errors
	InternalError = New("INTERNAL_ERROR", "An unexpected error occurred", http.StatusInternalServerError)
)
