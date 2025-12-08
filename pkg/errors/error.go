// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"encoding/json"
	"errors"

	"github.com/absmach/supermq/pkg/errors/codes"
)

// Error is a structured error with a code, context, and optional cause.
// It implements Go's error interface and supports unwrapping via errors.Is/As.
//
// Create errors using the E() factory:
//
//	E(codes.NotFound).With(KeyEntityType, "client").Wrap(dbErr)
//	E(codes.ValidationFailed).New("invalid email format")
type Error struct {
	code    codes.Code
	msg     string
	context *ErrorContext
	cause   error
}

// Error returns the full error message including cause chain.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return e.msg + ": " + e.cause.Error()
	}
	return e.msg
}

// Unwrap returns the underlying cause for errors.Is/As support.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Code returns the error code.
func (e *Error) Code() codes.Code {
	if e == nil {
		return codes.Code{}
	}
	return e.code
}

// Msg returns just this error's message without the cause.
func (e *Error) Msg() string {
	if e == nil {
		return ""
	}
	return e.msg
}

// Context returns the error context.
func (e *Error) Context() *ErrorContext {
	if e == nil {
		return nil
	}
	return e.context
}

// With adds a context value. Returns a new Error (immutable).
func (e *Error) With(key ContextKey, value string) *Error {
	if e == nil {
		return nil
	}
	newCtx := e.context.clone()
	newCtx.set(key.Name(), value, key.Public())
	return &Error{
		code:    e.code,
		msg:     e.msg,
		context: newCtx,
		cause:   e.cause,
	}
}

// WithPrivate adds a private context value (for logging only).
func (e *Error) WithPrivate(key string, value any) *Error {
	if e == nil {
		return nil
	}
	newCtx := e.context.clone()
	newCtx.set(key, value, false)
	return &Error{
		code:    e.code,
		msg:     e.msg,
		context: newCtx,
		cause:   e.cause,
	}
}

// Wrap wraps another error as the cause.
func (e *Error) Wrap(cause error) *Error {
	if e == nil {
		return nil
	}
	return &Error{
		code:    e.code,
		msg:     e.msg,
		context: e.context,
		cause:   cause,
	}
}

// MarshalJSON returns the JSON representation for API responses.
// Only public context values allowed by the code are included.
func (e *Error) MarshalJSON() ([]byte, error) {
	if e == nil {
		return json.Marshal(map[string]string{"message": ""})
	}

	resp := map[string]any{
		"code":    e.code.ID(),
		"message": e.code.Message(),
	}

	if e.context != nil {
		if details := e.context.PublicFiltered(e.code.Expose()); details != nil {
			resp["details"] = details
		}
	}

	return json.Marshal(resp)
}

// LogContext returns all context values for logging purposes.
func (e *Error) LogContext() map[string]any {
	if e == nil || e.context == nil {
		return nil
	}
	return e.context.All()
}

// --- Factory ---

// E creates an ErrorBuilder for the given code.
// This is the primary entry point for creating errors.
//
//	E(codes.NotFound).Wrap(err)
//	E(codes.CreateFailed).With(KeyEntityType, "client").Wrap(err)
//	E(codes.CreateFailed).With(KeyClientIDs, strings.Join(ids, ",")).Wrap(err)
func E(code codes.Code) *ErrorBuilder {
	return &ErrorBuilder{code: code}
}

// ErrorBuilder provides a fluent API for building errors.
type ErrorBuilder struct {
	code    codes.Code
	msg     string
	context *ErrorContext
}

// With adds a context value to the builder.
func (b *ErrorBuilder) With(key ContextKey, value string) *ErrorBuilder {
	if b.context == nil {
		b.context = newContext()
	}
	b.context.set(key.Name(), value, key.Public())
	return b
}

// WithPrivate adds a private context value to the builder (for logging only).
func (b *ErrorBuilder) WithPrivate(key string, value any) *ErrorBuilder {
	if b.context == nil {
		b.context = newContext()
	}
	b.context.set(key, value, false)
	return b
}

// Msg sets a custom log message (different from the code's user-facing message).
func (b *ErrorBuilder) Msg(msg string) *ErrorBuilder {
	b.msg = msg
	return b
}

// Wrap creates the error with the given cause.
func (b *ErrorBuilder) Wrap(cause error) *Error {
	msg := b.msg
	if msg == "" {
		msg = b.code.Message()
	}
	return &Error{
		code:    b.code,
		msg:     msg,
		context: b.context,
		cause:   cause,
	}
}

// New creates the error with a custom message and no cause.
func (b *ErrorBuilder) New(msg string) *Error {
	return &Error{
		code:    b.code,
		msg:     msg,
		context: b.context,
	}
}

// Err creates the error using the code's default message.
func (b *ErrorBuilder) Err() *Error {
	msg := b.msg
	if msg == "" {
		msg = b.code.Message()
	}
	return &Error{
		code:    b.code,
		msg:     msg,
		context: b.context,
	}
}

// --- Standard library wrappers ---

// Is reports whether any error in err's tree matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's tree that matches target.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Unwrap returns the wrapper and the underlying error by separating them.
// Returns (wrapper, underlying) where wrapper is this error's message
// and underlying is the cause.
func Unwrap(err error) (error, error) {
	if e, ok := err.(*Error); ok {
		if e.cause == nil {
			return nil, New(e.msg)
		}
		return New(e.msg), e.cause
	}

	// For standard library errors, return nil wrapper
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		return err, unwrapped
	}
	return nil, err
}

// GetCode extracts the error code from an error chain.
// Returns zero Code if no *Error is found.
func GetCode(err error) codes.Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code()
	}
	return codes.Code{}
}

// GetContext extracts the error context from an error chain.
// Returns nil if no *Error is found.
func GetContext(err error) *ErrorContext {
	var e *Error
	if errors.As(err, &e) {
		return e.Context()
	}
	return nil
}

// StatusCode returns the HTTP status code for an error.
// Returns 500 Internal Server Error if the error has no code.
func StatusCode(err error) int {
	code := GetCode(err)
	if code.IsZero() {
		return 500
	}
	return code.StatusCode()
}

// --- Legacy API for backward compatibility ---

// Legacy error values.
var (
	ErrMalformedEntity         = New("malformed entity specification")
	ErrUnsupportedContentType  = New("invalid content type")
	ErrUnidentified            = New("unidentified error")
	ErrEmptyPath               = New("empty file path")
	ErrStatusAlreadyAssigned   = New("status already assigned")
	ErrRollbackTx              = New("failed to rollback transaction")
	ErrAuthentication          = New("failed to perform authentication over the entity")
	ErrAuthorization           = New("failed to perform authorization over the entity")
	ErrMissingDomainMember     = New("member id is not member of domain")
	ErrMissingMember           = New("member id is not found")
	ErrEmailAlreadyExists      = New("email id already exists")
	ErrUsernameNotAvailable    = New("username not available")
	ErrDomainRouteNotAvailable = New("domain route not available")
	ErrChannelRouteNotAvailable = New("channel route not available")
	ErrTryAgain                = New("Something went wrong, please try again")
	ErrRouteNotAvailable       = New("route not available")
)

// New creates a simple error with the given message.
func New(text string) error {
	return &Error{msg: text}
}

// Wrap wraps err with the wrapper error message.
// Returns nil if wrapper is nil.
func Wrap(wrapper, err error) error {
	if wrapper == nil {
		return nil
	}
	if err == nil {
		return wrapper
	}

	// If wrapper is already our Error type, preserve its structure
	if e, ok := wrapper.(*Error); ok {
		return &Error{
			code:    e.code,
			msg:     e.msg,
			context: e.context,
			cause:   err,
		}
	}

	return &Error{
		msg:   wrapper.Error(),
		cause: err,
	}
}

// Contains checks if e2 is contained anywhere in e1's error chain.
func Contains(e1, e2 error) bool {
	if e1 == nil || e2 == nil {
		return e1 == e2
	}

	// Check if messages match
	if e1.Error() == e2.Error() {
		return true
	}

	// Check our Error type
	if e, ok := e1.(*Error); ok {
		if e.msg == e2.Error() {
			return true
		}
		if e.cause != nil {
			return Contains(e.cause, e2)
		}
	}

	// Check standard unwrap
	if unwrapped := errors.Unwrap(e1); unwrapped != nil {
		return Contains(unwrapped, e2)
	}

	return false
}
