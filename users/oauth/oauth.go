// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package oauth provides OAuth2 authentication implementation for users service.
// It handles both web-based OAuth flow and device authorization flow for CLI clients.
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"strings"

	"github.com/absmach/supermq/oauth"
	"github.com/absmach/supermq/pkg/oauth2"
	goauth2 "golang.org/x/oauth2"
)

// Re-export constants from oauth package for backward compatibility.
const (
	DeviceCodeLength      = oauth.DeviceCodeLength
	DeviceCodePollTimeout = oauth.DeviceCodePollTimeout
	CodeCheckInterval     = oauth.CodeCheckInterval
	DeviceStatePrefix     = oauth.DeviceStatePrefix
)

// Re-export errors from oauth package for backward compatibility.
var (
	ErrDeviceCodeExpired   = oauth.ErrDeviceCodeExpired
	ErrDeviceCodePending   = oauth.ErrDeviceCodePending
	ErrSlowDown            = oauth.ErrSlowDown
	ErrAccessDenied        = oauth.ErrAccessDenied
	ErrInvalidState        = oauth.ErrInvalidState
	ErrEmptyCode           = oauth.ErrEmptyCode
	ErrInvalidProvider     = oauth.ErrInvalidProvider
	ErrDeviceCodeNotFound  = oauth.ErrDeviceCodeNotFound
	ErrUserCodeNotFound    = oauth.ErrUserCodeNotFound
)

// Service is an alias for oauth.Service for backward compatibility.
// This allows existing code to continue using users/oauth.Service.
type Service = oauth.Service

// generateUserCode generates a human-friendly code like "ABCD-EFGH".
func generateUserCode() (string, error) {
	b := make([]byte, DeviceCodeLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	code = strings.ToUpper(code[:DeviceCodeLength])
	// Format as XXXX-XXXX
	if len(code) >= 8 {
		code = code[:4] + "-" + code[4:8]
	}
	return code, nil
}

// generateDeviceCode generates a random device code.
func generateDeviceCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// IsDeviceFlowState checks if the state parameter indicates a device flow.
func IsDeviceFlowState(state string) bool {
	return strings.HasPrefix(state, DeviceStatePrefix)
}

// ExtractUserCodeFromState extracts the user code from a device flow state.
func ExtractUserCodeFromState(state string) string {
	return strings.TrimPrefix(state, DeviceStatePrefix)
}

// ExchangeCode exchanges an authorization code for an access token.
// If redirectURL is provided, it uses ExchangeWithRedirect, otherwise uses Exchange.
func ExchangeCode(ctx context.Context, provider oauth2.Provider, code, redirectURL string) (goauth2.Token, error) {
	if redirectURL != "" {
		return provider.ExchangeWithRedirect(ctx, code, redirectURL)
	}
	return provider.Exchange(ctx, code)
}
