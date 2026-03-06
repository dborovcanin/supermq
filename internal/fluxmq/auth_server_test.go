// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	authv1 "github.com/absmach/fluxmq/pkg/proto/auth/v1"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	chmocks "github.com/absmach/supermq/channels/mocks"
	clmocks "github.com/absmach/supermq/clients/mocks"
	dmocks "github.com/absmach/supermq/domains/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	testClientID     = "client-1"
	testDomainID     = "11111111-1111-1111-1111-111111111111"
	testChannelID    = "22222222-2222-2222-2222-222222222222"
	testDomainRoute  = "domain-route"
	testChannelRoute = "channel-route"
	testExternalID   = "33333333-3333-3333-3333-333333333333"
	testUsername     = "user@example.com"
	testPassword     = "secret"
)

var silentLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func newTestServer(t *testing.T) (*AuthServer, *clmocks.ClientsServiceClient, *chmocks.ChannelsServiceClient, *dmocks.DomainsServiceClient) {
	clients := clmocks.NewClientsServiceClient(t)
	channels := chmocks.NewChannelsServiceClient(t)
	domains := dmocks.NewDomainsServiceClient(t)

	srv := NewAuthServer(AuthServerConfig{
		Clients:    clients,
		Channels:   channels,
		Domains:    domains,
		ClientType: "client",
		Logger:     silentLogger,
	})

	return srv, clients, channels, domains
}

func TestAuthenticate_Success(t *testing.T) {
	srv, clients, _, _ := newTestServer(t)

	clients.On("Authenticate", mock.Anything, mock.AnythingOfType("*v1.AuthnReq")).
		Return(&grpcClientsV1.AuthnRes{Authenticated: true, Id: testExternalID}, nil)

	res, err := srv.Authenticate(context.Background(), connect.NewRequest(&authv1.AuthnReq{
		ClientId: testClientID,
		Username: testUsername,
		Password: testPassword,
	}))

	require.NoError(t, err)
	assert.True(t, res.Msg.Authenticated)
	assert.Equal(t, testExternalID, res.Msg.Id)
	assert.Equal(t, uint32(0), res.Msg.ReasonCode)
}

func TestAuthenticate_BadCredentials(t *testing.T) {
	srv, clients, _, _ := newTestServer(t)

	clients.On("Authenticate", mock.Anything, mock.AnythingOfType("*v1.AuthnReq")).
		Return(&grpcClientsV1.AuthnRes{Authenticated: false}, nil)

	res, err := srv.Authenticate(context.Background(), connect.NewRequest(&authv1.AuthnReq{
		ClientId: testClientID,
		Username: testUsername,
		Password: "wrong",
	}))

	require.NoError(t, err)
	assert.False(t, res.Msg.Authenticated)
	assert.Equal(t, uint32(2), res.Msg.ReasonCode)
	assert.Contains(t, res.Msg.Reason, "bad credentials")
}

func TestAuthenticate_RPCError(t *testing.T) {
	srv, clients, _, _ := newTestServer(t)

	clients.On("Authenticate", mock.Anything, mock.AnythingOfType("*v1.AuthnReq")).
		Return(nil, status.Error(codes.Unavailable, "service down"))

	res, err := srv.Authenticate(context.Background(), connect.NewRequest(&authv1.AuthnReq{
		ClientId: testClientID,
		Username: testUsername,
		Password: testPassword,
	}))

	require.NoError(t, err)
	assert.False(t, res.Msg.Authenticated)
	assert.Equal(t, uint32(1), res.Msg.ReasonCode)
}

func TestAuthorize_PublishWithUUIDs(t *testing.T) {
	srv, _, channels, _ := newTestServer(t)

	channels.On("Authorize", mock.Anything, mock.MatchedBy(func(req *grpcChannelsV1.AuthzReq) bool {
		return req.DomainId == testDomainID &&
			req.ChannelId == testChannelID &&
			req.ClientId == testExternalID &&
			req.Type == publishConnType
	})).Return(&grpcChannelsV1.AuthzRes{Authorized: true}, nil)

	topic := "m/" + testDomainID + "/c/" + testChannelID + "/temp"
	res, err := srv.Authorize(context.Background(), connect.NewRequest(&authv1.AuthzReq{
		ExternalId: testExternalID,
		Topic:      topic,
		Action:     authv1.Action_ACTION_PUBLISH,
	}))

	require.NoError(t, err)
	assert.True(t, res.Msg.Authorized)
}

func TestAuthorize_SubscribeWithUUIDs(t *testing.T) {
	srv, _, channels, _ := newTestServer(t)

	channels.On("Authorize", mock.Anything, mock.MatchedBy(func(req *grpcChannelsV1.AuthzReq) bool {
		return req.Type == subscribeConnType
	})).Return(&grpcChannelsV1.AuthzRes{Authorized: true}, nil)

	topic := "m/" + testDomainID + "/c/" + testChannelID
	res, err := srv.Authorize(context.Background(), connect.NewRequest(&authv1.AuthzReq{
		ExternalId: testExternalID,
		Topic:      topic,
		Action:     authv1.Action_ACTION_SUBSCRIBE,
	}))

	require.NoError(t, err)
	assert.True(t, res.Msg.Authorized)
}

func TestAuthorize_Denied(t *testing.T) {
	srv, _, channels, _ := newTestServer(t)

	channels.On("Authorize", mock.Anything, mock.Anything).
		Return(&grpcChannelsV1.AuthzRes{Authorized: false}, nil)

	topic := "m/" + testDomainID + "/c/" + testChannelID
	res, err := srv.Authorize(context.Background(), connect.NewRequest(&authv1.AuthzReq{
		ExternalId: testExternalID,
		Topic:      topic,
		Action:     authv1.Action_ACTION_PUBLISH,
	}))

	require.NoError(t, err)
	assert.False(t, res.Msg.Authorized)
	assert.Equal(t, uint32(4), res.Msg.ReasonCode)
}

func TestAuthorize_NonSuperMQTopic(t *testing.T) {
	srv, _, _, _ := newTestServer(t)

	res, err := srv.Authorize(context.Background(), connect.NewRequest(&authv1.AuthzReq{
		ExternalId: testExternalID,
		Topic:      "sensors/temp/room1",
		Action:     authv1.Action_ACTION_PUBLISH,
	}))

	require.NoError(t, err)
	assert.True(t, res.Msg.Authorized)
}

func TestAuthorize_MalformedTopic(t *testing.T) {
	srv, _, _, _ := newTestServer(t)

	res, err := srv.Authorize(context.Background(), connect.NewRequest(&authv1.AuthzReq{
		ExternalId: testExternalID,
		Topic:      "m/domain-only",
		Action:     authv1.Action_ACTION_PUBLISH,
	}))

	require.NoError(t, err)
	assert.False(t, res.Msg.Authorized)
	assert.Equal(t, uint32(1), res.Msg.ReasonCode)
}

func TestAuthorize_RouteResolution(t *testing.T) {
	srv, _, channels, domains := newTestServer(t)

	domains.On("RetrieveIDByRoute", mock.Anything, mock.MatchedBy(func(req *grpcCommonV1.RetrieveIDByRouteReq) bool {
		return req.Route == testDomainRoute
	})).Return(&grpcCommonV1.RetrieveEntityRes{
		Entity: &grpcCommonV1.EntityBasic{Id: testDomainID},
	}, nil)

	channels.On("RetrieveIDByRoute", mock.Anything, mock.MatchedBy(func(req *grpcCommonV1.RetrieveIDByRouteReq) bool {
		return req.Route == testChannelRoute && req.DomainId == testDomainID
	})).Return(&grpcCommonV1.RetrieveEntityRes{
		Entity: &grpcCommonV1.EntityBasic{Id: testChannelID},
	}, nil)

	channels.On("Authorize", mock.Anything, mock.MatchedBy(func(req *grpcChannelsV1.AuthzReq) bool {
		return req.DomainId == testDomainID && req.ChannelId == testChannelID
	})).Return(&grpcChannelsV1.AuthzRes{Authorized: true}, nil)

	topic := "m/" + testDomainRoute + "/c/" + testChannelRoute + "/data"
	res, err := srv.Authorize(context.Background(), connect.NewRequest(&authv1.AuthzReq{
		ExternalId: testExternalID,
		Topic:      topic,
		Action:     authv1.Action_ACTION_PUBLISH,
	}))

	require.NoError(t, err)
	assert.True(t, res.Msg.Authorized)
}

func TestAuthorize_RouteResolutionCached(t *testing.T) {
	srv, _, channels, domains := newTestServer(t)

	// First call resolves routes
	domains.On("RetrieveIDByRoute", mock.Anything, mock.Anything).
		Return(&grpcCommonV1.RetrieveEntityRes{
			Entity: &grpcCommonV1.EntityBasic{Id: testDomainID},
		}, nil).Once()

	channels.On("RetrieveIDByRoute", mock.Anything, mock.Anything).
		Return(&grpcCommonV1.RetrieveEntityRes{
			Entity: &grpcCommonV1.EntityBasic{Id: testChannelID},
		}, nil).Once()

	channels.On("Authorize", mock.Anything, mock.Anything).
		Return(&grpcChannelsV1.AuthzRes{Authorized: true}, nil)

	topic := "m/" + testDomainRoute + "/c/" + testChannelRoute
	req := connect.NewRequest(&authv1.AuthzReq{
		ExternalId: testExternalID,
		Topic:      topic,
		Action:     authv1.Action_ACTION_PUBLISH,
	})

	// First call — resolves routes
	res, err := srv.Authorize(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, res.Msg.Authorized)

	// Second call — uses cache, no additional RetrieveIDByRoute calls
	res, err = srv.Authorize(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, res.Msg.Authorized)

	// Verify route resolution was called only once
	domains.AssertNumberOfCalls(t, "RetrieveIDByRoute", 1)
	channels.AssertNumberOfCalls(t, "RetrieveIDByRoute", 1)
}

func TestAuthorize_RPCError(t *testing.T) {
	srv, _, channels, _ := newTestServer(t)

	channels.On("Authorize", mock.Anything, mock.Anything).
		Return(nil, status.Error(codes.Internal, "internal error"))

	topic := "m/" + testDomainID + "/c/" + testChannelID
	res, err := srv.Authorize(context.Background(), connect.NewRequest(&authv1.AuthzReq{
		ExternalId: testExternalID,
		Topic:      topic,
		Action:     authv1.Action_ACTION_PUBLISH,
	}))

	require.NoError(t, err)
	assert.False(t, res.Msg.Authorized)
	assert.Equal(t, uint32(3), res.Msg.ReasonCode)
}

func TestAuthorize_WildcardsInDomainChannel(t *testing.T) {
	srv, _, _, _ := newTestServer(t)

	res, err := srv.Authorize(context.Background(), connect.NewRequest(&authv1.AuthzReq{
		ExternalId: testExternalID,
		Topic:      "m/domain+/c/channel#",
		Action:     authv1.Action_ACTION_SUBSCRIBE,
	}))

	require.NoError(t, err)
	assert.False(t, res.Msg.Authorized)
	assert.Equal(t, uint32(1), res.Msg.ReasonCode)
}
