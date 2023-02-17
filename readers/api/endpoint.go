// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/readers"
)

func listMessagesEndpoint(svc readers.MessageRepository, tc mainflux.ThingsServiceClient, ac mainflux.AuthServiceClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		fmt.Println()
		fmt.Println("Just got into listMessagesEndpoint from transport -> endpoint")
		fmt.Println()

		req := request.(listMessagesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}
		fmt.Println()
		fmt.Println("Passed req.validate")
		fmt.Println()

		if err := authorize(ctx, req, tc, ac); err != nil {
			return nil, errors.Wrap(errors.ErrAuthorization, err)
		}
		fmt.Println()
		fmt.Println("Passed authorize, Calling svc.Readall()")
		fmt.Println()
		page, err := svc.ReadAll(req.chanID, req.pageMeta)
		if err != nil {
			fmt.Println()
			fmt.Println("Got error in svc.Readll : ", err)
			fmt.Println()
			return nil, err
		}
		fmt.Println()
		fmt.Println("We now got this page :- \n\n : ", page)
		fmt.Println()

		return pageRes{
			PageMetadata: page.PageMetadata,
			Total:        page.Total,
			Messages:     page.Messages,
		}, nil
	}
}
