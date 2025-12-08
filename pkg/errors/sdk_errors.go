// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/absmach/supermq/pkg/errors/codes"
)

// SDKError is an error type for SuperMQ SDK.
type SDKError interface {
	error
	StatusCode() int
	Unwrap() error
}

var _ SDKError = (*sdkError)(nil)

// sdkError is the concrete implementation of SDKError.
type sdkError struct {
	err        error
	statusCode int
}

// NewSDKError creates an SDK error from any error.
func NewSDKError(err error) SDKError {
	if err == nil {
		return nil
	}
	return &sdkError{err: err}
}

// NewSDKErrorWithStatus creates an SDK error with a specific status code.
func NewSDKErrorWithStatus(err error, statusCode int) SDKError {
	if err == nil {
		return nil
	}
	return &sdkError{err: err, statusCode: statusCode}
}

// Error returns the error message.
func (e *sdkError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	if e.statusCode != 0 {
		return fmt.Sprintf("Status: %s: %s", http.StatusText(e.statusCode), e.err.Error())
	}
	return e.err.Error()
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *sdkError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// StatusCode returns the HTTP status code.
func (e *sdkError) StatusCode() int {
	if e == nil {
		return 0
	}
	return e.statusCode
}

// Code returns the error code if the underlying error has one.
func (e *sdkError) Code() codes.Code {
	if e == nil || e.err == nil {
		return codes.Code{}
	}
	var err *Error
	if As(e.err, &err) {
		return err.Code()
	}
	return codes.Code{}
}

// MarshalJSON returns the JSON representation.
func (e *sdkError) MarshalJSON() ([]byte, error) {
	if e == nil || e.err == nil {
		return json.Marshal(map[string]string{
			"message": http.StatusText(e.statusCode),
		})
	}

	// If underlying error is our Error type, use its marshaling
	var err *Error
	if As(e.err, &err) {
		return err.MarshalJSON()
	}

	// Fallback for other error types
	return json.Marshal(map[string]string{
		"message": e.err.Error(),
	})
}

// apiErrorResponse represents the JSON structure from API error responses.
type apiErrorResponse struct {
	Code    string         `json:"code,omitempty"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// CheckError checks the HTTP response and returns an SDK error if needed.
func CheckError(resp *http.Response, expectedStatusCodes ...int) SDKError {
	if resp == nil {
		return nil
	}

	for _, code := range expectedStatusCodes {
		if resp.StatusCode == code {
			return nil
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewSDKErrorWithStatus(
			E(codes.InternalError).Wrap(err),
			resp.StatusCode,
		)
	}

	var content apiErrorResponse
	if err := json.Unmarshal(body, &content); err != nil {
		return NewSDKErrorWithStatus(
			E(codes.InternalError).New(err.Error()),
			resp.StatusCode,
		)
	}

	// Build error from response
	var resultErr *Error
	if content.Code != "" {
		code := codes.New(content.Code, content.Message, resp.StatusCode)
		resultErr = E(code).New(content.Message)
	} else {
		resultErr = E(codes.InternalError).New(content.Message)
	}

	return NewSDKErrorWithStatus(resultErr, resp.StatusCode)
}
