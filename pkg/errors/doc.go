// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

/*
Package errors provides structured error handling for SuperMQ.

# Creating Errors

Use the E() factory with a code from the codes package:

	import (
		"github.com/absmach/supermq/pkg/errors"
		"github.com/absmach/supermq/pkg/errors/codes"
	)

	// Basic error
	err := errors.E(codes.NotFound).Err()

	// Error with cause
	err := errors.E(codes.NotFound).Wrap(dbErr)

	// Error with custom message
	err := errors.E(codes.ValidationFailed).New("email is required")

	// Error with context (fluent builder)
	err := errors.E(codes.NotFound).
		With(errors.KeyEntityType, "client").
		With(errors.KeyEntityID, "123").
		Wrap(dbErr)

# Context Keys

Context keys can be public (exposable to API) or private (logging only):

	// Public keys - can be exposed in API responses
	errors.KeyEntityType  // "entity_type"
	errors.KeyEntityID    // "entity_id"
	errors.KeyField       // "field"

	// Private keys - for logging only
	errors.KeyPGCode      // "pg_code"
	errors.KeyQuery       // "query"

# JSON Serialization

Errors serialize to JSON with double-filtering: a context value is only included
if BOTH the key is safe AND the code's expose list includes it.

	// Code defines what context keys can be exposed
	NotFound = codes.New("NOT_FOUND", "Resource not found", 404, "entity_type", "entity_id")

	// Only entity_type and entity_id will appear in JSON, not pg_code
	err := errors.E(codes.NotFound).
		With(errors.KeyEntityType, "client").   // exposed (public key + in expose list)
		WithPrivate("pg_code", "23505").        // not exposed (private)
		Wrap(dbErr)

# Go Error Compatibility

Errors support Go's standard error handling:

	// errors.Is works with the cause chain
	if errors.Is(err, sql.ErrNoRows) { ... }

	// errors.As extracts typed errors
	var e *errors.Error
	if errors.As(err, &e) {
		code := e.Code()
	}

	// Helper functions
	code := errors.GetCode(err)
	status := errors.StatusCode(err)
	ctx := errors.GetContext(err)

# Service/Repository Packages

Import layer-specific codes for convenience:

	import svcerr "github.com/absmach/supermq/pkg/errors/service"
	import repoerr "github.com/absmach/supermq/pkg/errors/repository"

	return errors.E(svcerr.NotFound).Wrap(err)
	return errors.E(repoerr.CreateFailed).Wrap(err)
*/
package errors
