// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"encoding/json"
)

// Error specifies an API that must be fullfiled by error type.
type Error interface {
	// Error implements the error interface.
	Error() string

	// Msg returns error message.
	Msg() string

	// Unwrap returns wrapped error.
	Unwrap() error

	// MarshalJSON returns a marshaled error.
	MarshalJSON() ([]byte, error)
}

var _ Error = (*CustomError)(nil)

// CustomError represents a Magistrala error.
type CustomError struct {
	Details string
	Err     error
}

// New returns an Error that formats as the given text.
func New(text string) Error {
	return &CustomError{
		Details: text,
		Err:     nil,
	}
}

// New returns an Error that formats as the given text.
func NewCustomError(details string, err error) *CustomError {
	return &CustomError{
		Details: details,
		Err:     err,
	}
}

func NewError(text string, err error) Error {
	return &CustomError{
		Details: text,
		Err:     Cast(err),
	}
}

func (ce *CustomError) Error() string {
	if ce == nil {
		return ""
	}
	if ce.Err == nil {
		return ce.Details
	}
	return ce.Details + " : " + ce.Err.Error()
}

func (ce *CustomError) Msg() string {
	return ce.Details
}

func (ce *CustomError) Unwrap() error {
	return ce.Err
}

func (ce *CustomError) MarshalJSON() ([]byte, error) {
	var val string
	if e := ce.Unwrap(); e != nil {
		val = e.Error()
	}
	return json.Marshal(&struct {
		Err string `json:"error"`
		Msg string `json:"message"`
	}{
		Err: val,
		Msg: ce.Msg(),
	})
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
		return Contains(ce.Unwrap(), e2)
	}
	return e1.Error() == e2.Error()
}

// Wrap returns an Error that wrap err with wrapper.
func Wrap(wrapper, err error) Error {
	if wrapper == nil || err == nil {
		return wrapper.(Error)
	}
	if w, ok := wrapper.(Error); ok {
		return &CustomError{
			Details: w.Msg(),
			Err:     Cast(err),
		}
	}
	return &CustomError{
		Details: wrapper.Error(),
		Err:     Cast(err),
	}
}

// Unwrap returns the wrapper and the error by separating the Wrapper from the error.
func Unwrap(err error) (Error, error) {
	if ce, ok := err.(Error); ok {
		if ce.Unwrap() == nil {
			return nil, New(ce.Msg())
		}
		return New(ce.Msg()), ce.Unwrap()
	}

	return nil, Cast(err)
}

// Cast returns an Error from an error.
func Cast(err error) Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(Error); ok {
		return e
	}
	return &CustomError{
		Details: err.Error(),
		Err:     nil,
	}
}
