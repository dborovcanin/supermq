// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrInvalidKeyIssuedAt indicates that the Key is being used before it's issued.
	ErrInvalidKeyIssuedAt = errors.New("invalid issue time")

	// ErrKeyExpired indicates that the Key is expired.
	ErrKeyExpired = errors.New("use of expired key")

	// ErrAPIKeyExpired indicates that the Key is expired
	// and that the key type is API key.
	ErrAPIKeyExpired = errors.New("use of expired API key")
)

type KeyType uint32

func (kt KeyType) String() string {
	switch kt {
	case AccessKey:
		return "access"
	case RefreshKey:
		return "refresh"
	case RecoveryKey:
		return "recovery"
	case APIKey:
		return "API"
	default:
		return "unknown"
	}
}

const (
	// AccessKey is temporary User key received on successfull login.
	AccessKey KeyType = iota
	// RefreshKey is a temporary User key used to generate a new access key.
	RefreshKey
	// RecoveryKey represents a key for resseting password.
	RecoveryKey
	// APIKey enables the one to act on behalf of the user.
	APIKey
)

// Key represents API key.
type Key struct {
	ID        string    `json:"id,omitempty"`
	Type      KeyType   `json:"type,omitempty"`
	Issuer    string    `json:"issuer,omitempty"`
	SubjectID string    `json:"subject_id,omitempty"` // internal ID in our system
	Subject   string    `json:"subject,omitempty"`    // email or username or other unique identifier
	IssuedAt  time.Time `json:"issued_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func (key Key) String() string {
	return fmt.Sprintf(`{
	id: %s,
	type: %s,
	issuer_id: %s,
	subject_id: %s,
	subject: %s,
	iat: %v,
	eat: %v
}`, key.ID, key.Type, key.Issuer, key.SubjectID, key.Subject, key.IssuedAt, key.ExpiresAt)
}

type Token struct {
	Value string                 `json:"value,omitempty"`
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// Identity contains ID and Email.
type Identity struct {
	ID    string
	Email string
}

// Expired verifies if the key is expired.
func (k Key) Expired() bool {
	if k.Type == APIKey && k.ExpiresAt.IsZero() {
		return false
	}
	return k.ExpiresAt.UTC().Before(time.Now().UTC())
}

// KeyRepository specifies Key persistence API.
type KeyRepository interface {
	// Save persists the Key. A non-nil error is returned to indicate
	// operation failure
	Save(context.Context, Key) (string, error)

	// Retrieve retrieves Key by its unique identifier.
	Retrieve(context.Context, string, string) (Key, error)

	// Remove removes Key with provided ID.
	Remove(context.Context, string, string) error
}
