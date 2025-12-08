// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"encoding/json"

	"github.com/absmach/supermq/pkg/errors/codes"
)

// Error specifies an API that must be fulfilled by error type.
type Error interface {
	// Error implements the error interface.
	Error() string

	// Msg returns error message.
	Msg() string

	// Err returns wrapped error.
	Err() Error

	// Code returns the error code associated with this error.
	// Returns a zero Code if no code is set.
	Code() codes.Code

	// Context returns the error context containing structured metadata.
	// Returns nil if no context is set.
	Context() *ErrorContext

	// AllContext returns the merged context from the entire error chain.
	// This is useful for logging where you want all context values.
	AllContext() *ErrorContext

	// WithCode returns a new error with the given code.
	WithCode(code codes.Code) Error

	// WithContext returns a new error with the given key-value pair added to context.
	// Use the typed Set function for type-safe context setting.
	WithContext(key string, value any, safe bool) Error

	// MarshalJSON returns a marshaled error for API responses.
	// Only safe context values allowed by the error code are included.
	MarshalJSON() ([]byte, error)
}

var _ Error = (*customError)(nil)

// customError represents a SuperMQ error.
type customError struct {
	msg     string
	err     Error
	code    codes.Code
	context *ErrorContext
}

// New returns an Error that formats as the given text.
// The error has no code and no context.
// For errors with codes, use NewWithCode.
func New(text string) Error {
	return &customError{
		msg: text,
		err: nil,
	}
}

// NewWithCode returns an Error with the given code and message.
// The code's default message is used for API responses, while the provided
// message is used for internal logging.
func NewWithCode(code codes.Code, msg string) Error {
	return &customError{
		msg:  msg,
		code: code,
	}
}

func (ce *customError) Error() string {
	if ce == nil {
		return ""
	}
	if ce.err == nil {
		return ce.msg
	}
	return ce.msg + " : " + ce.err.Error()
}

func (ce *customError) Msg() string {
	if ce == nil {
		return ""
	}
	return ce.msg
}

func (ce *customError) Err() Error {
	if ce == nil {
		return nil
	}
	return ce.err
}

func (ce *customError) Code() codes.Code {
	if ce == nil {
		return codes.Code{}
	}
	// If this error has a code, return it
	if !ce.code.IsZero() {
		return ce.code
	}
	// Otherwise, try to inherit from wrapped error
	if ce.err != nil {
		return ce.err.Code()
	}
	return codes.Code{}
}

func (ce *customError) Context() *ErrorContext {
	if ce == nil {
		return nil
	}
	return ce.context
}

func (ce *customError) WithCode(code codes.Code) Error {
	if ce == nil {
		return &customError{code: code}
	}
	return &customError{
		msg:     ce.msg,
		err:     ce.err,
		code:    code,
		context: ce.context,
	}
}

func (ce *customError) WithContext(key string, value any, safe bool) Error {
	if ce == nil {
		newCtx := NewErrorContext()
		newCtx.entries[key] = contextEntry{value: value, safe: safe}
		return &customError{context: newCtx}
	}

	newCtx := ce.context.Clone()
	newCtx.entries[key] = contextEntry{value: value, safe: safe}

	return &customError{
		msg:     ce.msg,
		err:     ce.err,
		code:    ce.code,
		context: newCtx,
	}
}

// With is a convenience function to add a typed context value to an error.
// Returns the modified error.
func With[T any](err Error, key ContextKey[T], value T) Error {
	if err == nil {
		return nil
	}
	return err.WithContext(key.Name(), value, key.Safe())
}

// apiResponse is the structure returned in API responses for errors with codes.
type apiResponse struct {
	Code    string         `json:"code,omitempty"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// legacyResponse is the structure returned for backward compatibility
// with errors that don't have codes.
type legacyResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (ce *customError) MarshalJSON() ([]byte, error) {
	if ce == nil {
		return json.Marshal(legacyResponse{Error: "", Message: ""})
	}

	// Get code (either from this error or inherited from chain)
	code := ce.Code()

	if !code.IsZero() {
		// New behavior for errors with codes
		resp := apiResponse{
			Code:    code.ID(),
			Message: code.Message(),
		}

		// Get filtered safe context
		allContext := ce.AllContext()
		if allContext != nil {
			// Double filter: must be in code's expose list AND marked as safe
			resp.Details = filterContext(allContext, code.Expose())
		}

		return json.Marshal(resp)
	}

	// Legacy behavior for errors without codes
	// Maintains backward compatibility with existing tests
	var errMsg string
	if ce.err != nil {
		errMsg = ce.err.Msg()
	}
	return json.Marshal(legacyResponse{
		Error:   errMsg,
		Message: ce.msg,
	})
}

// filterContext returns only context entries that are in the allowed list.
// Note: The ErrorContext already filters for safe values, this adds
// the second filter based on what the code allows to expose.
func filterContext(ctx *ErrorContext, allowed []string) map[string]any {
	return ctx.SafeFiltered(allowed)
}

// AllContext collects context from the entire error chain.
// Values from outer errors take precedence over inner errors.
func (ce *customError) AllContext() *ErrorContext {
	if ce == nil {
		return nil
	}

	// Start with wrapped error's context (if any)
	var result *ErrorContext
	if ce.err != nil {
		result = ce.err.AllContext()
	}

	// Merge this error's context (this error's values take precedence)
	if ce.context != nil {
		if result == nil {
			result = ce.context.Clone()
		} else {
			result = result.Merge(ce.context)
		}
	}

	return result
}

// AllContext returns the merged context from the entire error chain.
// This is useful for logging where you want all context values.
func AllContext(err error) *ErrorContext {
	if err == nil {
		return nil
	}
	if ce, ok := err.(Error); ok {
		if cErr, ok := ce.(*customError); ok {
			return cErr.AllContext()
		}
	}
	return nil
}

// Contains inspects if e2 error is contained in any layer of e1 error.
func Contains(e1, e2 error) bool {
	if e1 == nil || e2 == nil {
		return e2 == e1
	}
	ce, ok := e1.(Error)
	if ok {
		if ce.Msg() == e2.Error() {
			return true
		}
		return Contains(ce.Err(), e2)
	}
	return e1.Error() == e2.Error()
}

// ContainsCode checks if the error chain contains an error with the given code.
func ContainsCode(err error, code codes.Code) bool {
	if err == nil || code.IsZero() {
		return false
	}
	ce, ok := err.(Error)
	if !ok {
		return false
	}
	if ce.Code().ID() == code.ID() {
		return true
	}
	if ce.Err() != nil {
		return ContainsCode(ce.Err(), code)
	}
	return false
}

// GetCode extracts the error code from an error.
// Returns a zero Code if the error has no code.
func GetCode(err error) codes.Code {
	if err == nil {
		return codes.Code{}
	}
	if ce, ok := err.(Error); ok {
		return ce.Code()
	}
	return codes.Code{}
}

// Wrap returns an Error that wraps err with wrapper.
// The wrapper's code is used for the resulting error.
// Context from both errors is merged, with wrapper's context taking precedence.
func Wrap(wrapper, err error) error {
	if wrapper == nil || err == nil {
		return wrapper
	}

	w := cast(wrapper)
	e := cast(err)

	// Merge context from wrapped error into wrapper's context
	var mergedCtx *ErrorContext
	if e != nil && e.Context() != nil {
		mergedCtx = e.Context().Clone()
	}
	if w.Context() != nil {
		if mergedCtx == nil {
			mergedCtx = w.Context().Clone()
		} else {
			mergedCtx = mergedCtx.Merge(w.Context())
		}
	}

	return &customError{
		msg:     w.Msg(),
		err:     e,
		code:    w.Code(),
		context: mergedCtx,
	}
}

// Unwrap returns the wrapper and the error by separating the Wrapper from the error.
func Unwrap(err error) (error, error) {
	if ce, ok := err.(Error); ok {
		if ce.Err() == nil {
			return nil, New(ce.Msg())
		}
		return New(ce.Msg()), ce.Err()
	}

	return nil, err
}

func cast(err error) Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(Error); ok {
		return e
	}
	return &customError{
		msg: err.Error(),
		err: nil,
	}
}
