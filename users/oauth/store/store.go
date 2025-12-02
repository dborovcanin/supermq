// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package store provides storage implementations for OAuth device codes.
package store

import (
	"github.com/absmach/supermq/oauth"
)

// Re-export constants from oauth package for backward compatibility.
const (
	DeviceCodeExpiry = oauth.DeviceCodeExpiry
)

// Re-export errors from oauth package for backward compatibility.
var (
	ErrDeviceCodeNotFound = oauth.ErrDeviceCodeNotFound
	ErrUserCodeNotFound   = oauth.ErrUserCodeNotFound
)

// DeviceCode is an alias for oauth.DeviceCode for backward compatibility.
type DeviceCode = oauth.DeviceCode

// DeviceCodeStore is an alias for oauth.DeviceCodeStore for backward compatibility.
type DeviceCodeStore = oauth.DeviceCodeStore
