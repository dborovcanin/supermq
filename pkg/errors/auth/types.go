package auth

import "github.com/absmach/magistrala/pkg/errors"

type (

	// AuthenticationError indicates failure occurred while authenticating the entity.
	AuthenticationError struct {
		*errors.CustomError
	}

	// AuthorizationError indicates failure occurred while authorizing the entity.
	AuthorizationError struct {
		*errors.CustomError
	}
)

func NewAuthNError(text string, err error) error {
	return &AuthenticationError{errors.NewErr(text, err)}
}

func NewAuthZError(text string, err error) error {
	return &AuthenticationError{errors.NewErr(text, err)}
}
