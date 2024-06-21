// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import "github.com/absmach/magistrala/pkg/errors"

type (
	ContentTypeError struct {
		*errors.CustomError
	}

	ValidationError struct {
		*errors.CustomError
	}

	InvalidParamsError struct {
		*errors.CustomError
	}
)
