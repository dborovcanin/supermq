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
)
