// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/http"
)

func sendMessageEndpoint(svc http.Service) endpoint.Endpoint {
	fmt.Println()
	fmt.Println("Got into http, sending message, sendMessageEndpoint")
	fmt.Println()
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(publishReq)
		fmt.Println()
		fmt.Println("Inside func returned by sendMessageEndpoint: req = \n", req)
		fmt.Println()

		if err := req.validate(); err != nil {
			return nil, err
		}

		fmt.Println()
		fmt.Println("validate func returned success, called svc.Publish")
		fmt.Println()
		return nil, svc.Publish(ctx, req.token, req.msg)
	}
}
