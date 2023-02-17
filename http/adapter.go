// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"
	"fmt"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/messaging"
)

// Service specifies coap service API.
type Service interface {
	// Publish Messssage
	Publish(ctx context.Context, token string, msg *messaging.Message) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher messaging.Publisher
	things    mainflux.ThingsServiceClient
}

// New instantiates the HTTP adapter implementation.
func New(publisher messaging.Publisher, things mainflux.ThingsServiceClient) Service {
	return &adapterService{
		publisher: publisher,
		things:    things,
	}
}

func (as *adapterService) Publish(ctx context.Context, token string, msg *messaging.Message) error {
	fmt.Println()
	fmt.Println("Inside svc.Publish")
	fmt.Println()
	ar := &mainflux.AccessByKeyReq{
		Token:  token,
		ChanID: msg.Channel,
	}
	thid, err := as.things.CanAccessByKey(ctx, ar)
	if err != nil {
		return err
	}
	msg.Publisher = thid.GetValue()
	fmt.Println()
	fmt.Println("Called as.pubslisher.publish")
	fmt.Println()

	return as.publisher.Publish(msg.Channel, msg)
}
