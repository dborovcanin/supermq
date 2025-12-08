// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package errors

// ContextKey is a key for error context values.
// Keys can be marked as public (exposable to API clients) or private (logging only).
type ContextKey struct {
	name   string
	public bool
}

// Name returns the key name.
func (k ContextKey) Name() string { return k.name }

// Public returns whether this key's value can be exposed to API clients.
func (k ContextKey) Public() bool { return k.public }

// NewPublicKey creates a context key whose value is safe to expose in API responses.
func NewPublicKey(name string) ContextKey {
	return ContextKey{name: name, public: true}
}

// NewPrivateKey creates a context key whose value is for internal logging only.
func NewPrivateKey(name string) ContextKey {
	return ContextKey{name: name, public: false}
}

// Predefined context keys.
var (
	// Public keys - can be exposed to API clients if the error code allows.
	KeyEntityType = NewPublicKey("entity_type")
	KeyEntityID   = NewPublicKey("entity_id")
	KeyOperation  = NewPublicKey("operation")
	KeyField      = NewPublicKey("field")
	KeyReason     = NewPublicKey("reason")
	KeyPermission = NewPublicKey("permission")
	KeyStatus     = NewPublicKey("status")

	// Private keys - for logging only, never exposed to clients.
	KeyDomainID     = NewPrivateKey("domain_id")
	KeyUserID       = NewPrivateKey("user_id")
	KeyPGCode       = NewPrivateKey("pg_code")
	KeyPGConstraint = NewPrivateKey("pg_constraint")
	KeyPGDetail     = NewPrivateKey("pg_detail")
	KeyQuery        = NewPrivateKey("query")
	KeyRequestID    = NewPrivateKey("request_id")
	KeyStackTrace   = NewPrivateKey("stack_trace")
	KeyClientIDs    = NewPrivateKey("client_ids")
	KeyChannelIDs   = NewPrivateKey("channel_ids")
)

// contextEntry stores a value along with its visibility flag.
type contextEntry struct {
	value  any
	public bool
}

// ErrorContext holds structured context information for errors.
type ErrorContext struct {
	entries map[string]contextEntry
}

func newContext() *ErrorContext {
	return &ErrorContext{entries: make(map[string]contextEntry)}
}

func (c *ErrorContext) set(key string, value any, public bool) {
	if c.entries == nil {
		c.entries = make(map[string]contextEntry)
	}
	c.entries[key] = contextEntry{value: value, public: public}
}

func (c *ErrorContext) clone() *ErrorContext {
	if c == nil {
		return newContext()
	}
	newCtx := newContext()
	for k, v := range c.entries {
		newCtx.entries[k] = v
	}
	return newCtx
}

// PublicFiltered returns only public values that are in the allowed list.
// This implements the double-filter: key must be public AND in the expose list.
func (c *ErrorContext) PublicFiltered(allowed []string) map[string]any {
	if c == nil || len(c.entries) == 0 || len(allowed) == 0 {
		return nil
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}

	result := make(map[string]any)
	for key, entry := range c.entries {
		if _, ok := allowedSet[key]; ok && entry.public {
			result[key] = entry.value
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// All returns all context values (for logging).
func (c *ErrorContext) All() map[string]any {
	if c == nil || len(c.entries) == 0 {
		return nil
	}
	result := make(map[string]any, len(c.entries))
	for k, v := range c.entries {
		result[k] = v.value
	}
	return result
}
