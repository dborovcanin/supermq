// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ mainflux.ThingsServiceClient = (*thingsServiceMock)(nil)

type thingsServiceMock struct {
	channels map[string]string
}

// NewThingsService returns mock implementation of things service.
func NewThingsService(channels map[string]string) mainflux.ThingsServiceClient {
	return &thingsServiceMock{channels}
}

func (svc thingsServiceMock) Identify(context.Context, *mainflux.Token, ...grpc.CallOption) (*mainflux.ThingID, error) {
	panic("not implemented")
}
func (svc thingsServiceMock) CanAccessByKey(ctx context.Context, in *mainflux.AccessByKeyReq, opts ...grpc.CallOption) (*mainflux.ThingID, error) {
	panic("not implemented")
}
func (svc thingsServiceMock) IsChannelOwner(ctx context.Context, in *mainflux.ChannelOwnerReq, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	panic("not implemented")
}
func (svc thingsServiceMock) CanAccessByID(ctx context.Context, in *mainflux.AccessByIDReq, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	panic("not implemented")
}
