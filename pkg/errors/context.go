// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package errors

import "fmt"

// ContextKey is a typed key for error context values.
// The type parameter T indicates the type of value associated with this key.
// Each key also knows whether its value is safe to expose in API responses.
type ContextKey[T any] struct {
	name string
	safe bool
}

// Name returns the string name of this context key.
func (k ContextKey[T]) Name() string { return k.name }

// Safe returns true if values for this key can be exposed in API responses.
func (k ContextKey[T]) Safe() bool { return k.safe }

// NewSafeKey creates a context key whose values are safe to expose in API responses.
func NewSafeKey[T any](name string) ContextKey[T] {
	return ContextKey[T]{name: name, safe: true}
}

// NewInternalKey creates a context key whose values should never be exposed in API responses.
// These values are only for internal logging and debugging.
func NewInternalKey[T any](name string) ContextKey[T] {
	return ContextKey[T]{name: name, safe: false}
}

// Predefined context keys for common error context values.
var (
	// Safe keys - can be exposed in API responses
	KeyEntityType = NewSafeKey[string]("entity_type")
	KeyEntityID   = NewSafeKey[string]("entity_id")
	KeyOperation  = NewSafeKey[string]("operation")
	KeyField      = NewSafeKey[string]("field")
	KeyReason     = NewSafeKey[string]("reason")

	// Internal keys - never exposed in API responses
	KeyDomainID     = NewInternalKey[string]("domain_id")
	KeyUserID       = NewInternalKey[string]("user_id")
	KeyPGCode       = NewInternalKey[string]("pg_code")
	KeyPGConstraint = NewInternalKey[string]("pg_constraint")
	KeyPGDetail     = NewInternalKey[string]("pg_detail")
	KeyQuery        = NewInternalKey[string]("query")
	KeyClientIDs    = NewInternalKey[[]string]("client_ids")
	KeyChannelIDs   = NewInternalKey[[]string]("channel_ids")
)

// contextEntry stores a value along with its safety flag.
type contextEntry struct {
	value any
	safe  bool
}

// ErrorContext holds structured context information for errors.
// It maintains type safety through the ContextKey mechanism and
// tracks which values are safe to expose in API responses.
type ErrorContext struct {
	entries map[string]contextEntry
}

// NewErrorContext creates a new empty error context.
func NewErrorContext() *ErrorContext {
	return &ErrorContext{
		entries: make(map[string]contextEntry),
	}
}

// Set adds a value to the context using a typed key.
// The safety flag is determined by the key definition.
func Set[T any](ctx *ErrorContext, key ContextKey[T], value T) {
	if ctx.entries == nil {
		ctx.entries = make(map[string]contextEntry)
	}
	ctx.entries[key.name] = contextEntry{
		value: value,
		safe:  key.safe,
	}
}

// Get retrieves a value from the context using a typed key.
// Returns the value and true if found, or zero value and false if not found.
func Get[T any](ctx *ErrorContext, key ContextKey[T]) (T, bool) {
	if ctx == nil || ctx.entries == nil {
		var zero T
		return zero, false
	}
	entry, ok := ctx.entries[key.name]
	if !ok {
		var zero T
		return zero, false
	}
	v, ok := entry.value.(T)
	return v, ok
}

// All returns all context entries as a map (for logging).
// This includes both safe and internal values.
func (ctx *ErrorContext) All() map[string]any {
	if ctx == nil || ctx.entries == nil {
		return nil
	}
	result := make(map[string]any, len(ctx.entries))
	for name, entry := range ctx.entries {
		result[name] = entry.value
	}
	return result
}

// Safe returns only context entries that are marked as safe to expose.
// This is used when preparing API responses.
func (ctx *ErrorContext) Safe() map[string]any {
	if ctx == nil || ctx.entries == nil {
		return nil
	}
	result := make(map[string]any)
	for name, entry := range ctx.entries {
		if entry.safe {
			result[name] = entry.value
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// SafeFiltered returns safe context entries filtered by the given allowed keys.
// Only entries that are both safe AND in the allowed list are returned.
// This provides defense in depth - even if a key is marked safe,
// each error code controls which keys it wants to expose.
func (ctx *ErrorContext) SafeFiltered(allowed []string) map[string]any {
	if ctx == nil || ctx.entries == nil || len(allowed) == 0 {
		return nil
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}

	result := make(map[string]any)
	for name, entry := range ctx.entries {
		if _, ok := allowedSet[name]; ok && entry.safe {
			result[name] = entry.value
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// Clone creates a deep copy of the context.
func (ctx *ErrorContext) Clone() *ErrorContext {
	if ctx == nil || ctx.entries == nil {
		return NewErrorContext()
	}
	newCtx := &ErrorContext{
		entries: make(map[string]contextEntry, len(ctx.entries)),
	}
	for k, v := range ctx.entries {
		newCtx.entries[k] = v
	}
	return newCtx
}

// Merge combines two contexts, with other's values taking precedence.
func (ctx *ErrorContext) Merge(other *ErrorContext) *ErrorContext {
	if ctx == nil && other == nil {
		return NewErrorContext()
	}
	if ctx == nil {
		return other.Clone()
	}
	if other == nil {
		return ctx.Clone()
	}

	newCtx := ctx.Clone()
	for k, v := range other.entries {
		newCtx.entries[k] = v
	}
	return newCtx
}

// String returns a string representation of the context for logging.
func (ctx *ErrorContext) String() string {
	if ctx == nil || ctx.entries == nil || len(ctx.entries) == 0 {
		return "{}"
	}
	return fmt.Sprintf("%v", ctx.All())
}
