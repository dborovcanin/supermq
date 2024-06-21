// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package repository

import (
	"github.com/absmach/magistrala/pkg/errors"
)

// Wrapper for Repository errors.
type (
	ConstraintError struct {
		*errors.CustomError
	}

	WriteError struct {
		*errors.CustomError
	}

	ReadError struct {
		*errors.CustomError
	}

	RollbackError struct {
		*errors.CustomError
	}

	TypeError struct {
		*errors.CustomError
	}

	OtherError struct {
		*errors.CustomError
	}

	NotFoundError struct {
		*errors.CustomError
	}
)

func NewConstraintError(text string, err error) *ConstraintError {
	return &ConstraintError{errors.NewErr(text, err)}
}

func NewWriteError(text string, err error) *WriteError {
	return &WriteError{errors.NewErr(text, err)}
}

func NewReadError(text string, err error) *ReadError {
	return &ReadError{errors.NewErr(text, err)}
}

func NewRollbackError(rbErr, err error) *RollbackError {
	return &RollbackError{errors.NewErr(rbErr.Error(), err)}
}

func NewTypeError(text string, err error) *TypeError {
	return &TypeError{errors.NewErr(text, err)}
}

func NewOtherError(text string, err error) *OtherError {
	return &OtherError{errors.NewErr(text, err)}
}

func NewNotFoundError(text string, err error) *NotFoundError {
	return &NotFoundError{errors.NewErr(text, err)}
}
