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

type errorRes struct {
	Code    string         `json:"code,omitempty"`
	Err     string         `json:"error"`
	Msg     string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Failed to read response body.
var errRespBody = New("failed to read response body")

// SDKError is an error type for SuperMQ SDK.
type SDKError interface {
	Error
	StatusCode() int
}

var _ SDKError = (*sdkError)(nil)

type sdkError struct {
	*customError
	statusCode int
}

func (se *sdkError) Error() string {
	if se == nil {
		return ""
	}
	if se.customError == nil {
		return http.StatusText(se.statusCode)
	}
	return fmt.Sprintf("Status: %s: %s", http.StatusText(se.statusCode), se.customError.Error())
}

func (se *sdkError) StatusCode() int {
	if se == nil {
		return 0
	}
	return se.statusCode
}

// Implement Error interface methods by delegating to customError
func (se *sdkError) Msg() string {
	if se == nil || se.customError == nil {
		return ""
	}
	return se.customError.Msg()
}

func (se *sdkError) Err() Error {
	if se == nil || se.customError == nil {
		return nil
	}
	return se.customError.Err()
}

func (se *sdkError) Code() codes.Code {
	if se == nil || se.customError == nil {
		return codes.Code{}
	}
	return se.customError.Code()
}

func (se *sdkError) Context() *ErrorContext {
	if se == nil || se.customError == nil {
		return nil
	}
	return se.customError.Context()
}

func (se *sdkError) WithCode(code codes.Code) Error {
	if se == nil {
		return &sdkError{
			customError: &customError{code: code},
			statusCode:  0,
		}
	}
	return &sdkError{
		customError: se.customError.WithCode(code).(*customError),
		statusCode:  se.statusCode,
	}
}

func (se *sdkError) WithContext(key string, value any, safe bool) Error {
	if se == nil {
		newCtx := NewErrorContext()
		newCtx.entries[key] = contextEntry{value: value, safe: safe}
		return &sdkError{
			customError: &customError{context: newCtx},
			statusCode:  0,
		}
	}
	return &sdkError{
		customError: se.customError.WithContext(key, value, safe).(*customError),
		statusCode:  se.statusCode,
	}
}

func (se *sdkError) MarshalJSON() ([]byte, error) {
	if se == nil || se.customError == nil {
		return json.Marshal(legacyResponse{Error: "", Message: http.StatusText(se.statusCode)})
	}
	return se.customError.MarshalJSON()
}

// NewSDKError returns an SDK Error that formats as the given text.
func NewSDKError(err error) SDKError {
	if err == nil {
		return nil
	}

	if e, ok := err.(Error); ok {
		return &sdkError{
			statusCode: 0,
			customError: &customError{
				msg:     e.Msg(),
				err:     cast(e.Err()),
				code:    e.Code(),
				context: e.Context(),
			},
		}
	}
	return &sdkError{
		customError: &customError{
			msg: err.Error(),
			err: nil,
		},
		statusCode: 0,
	}
}

// NewSDKErrorWithStatus returns an SDK Error setting the status code.
func NewSDKErrorWithStatus(err error, statusCode int) SDKError {
	if err == nil {
		return nil
	}

	if e, ok := err.(Error); ok {
		return &sdkError{
			statusCode: statusCode,
			customError: &customError{
				msg:     e.Msg(),
				err:     cast(e.Err()),
				code:    e.Code(),
				context: e.Context(),
			},
		}
	}
	return &sdkError{
		statusCode: statusCode,
		customError: &customError{
			msg: err.Error(),
			err: nil,
		},
	}
}

// CheckError will check the HTTP response status code and matches it with the given status codes.
// Since multiple status codes can be valid, we can pass multiple status codes to the function.
// The function then checks for errors in the HTTP response.
func CheckError(resp *http.Response, expectedStatusCodes ...int) SDKError {
	if resp == nil {
		return nil
	}

	for _, expectedStatusCode := range expectedStatusCodes {
		if resp.StatusCode == expectedStatusCode {
			return nil
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewSDKErrorWithStatus(Wrap(errRespBody, err), resp.StatusCode)
	}
	var content errorRes
	if err := json.Unmarshal(body, &content); err != nil {
		return NewSDKErrorWithStatus(err, resp.StatusCode)
	}

	// Build error from response
	var resultErr Error
	if content.Err == "" {
		resultErr = New(content.Msg)
	} else {
		resultErr = Wrap(New(content.Msg), New(content.Err)).(Error)
	}

	// If response had a code, attach it
	if content.Code != "" {
		// Create a custom code from the response
		// This preserves the code from the server response
		resultErr = resultErr.WithCode(codes.NewCode(
			content.Code,
			content.Msg,
			resp.StatusCode,
		))
	}

	return NewSDKErrorWithStatus(resultErr, resp.StatusCode)
}
