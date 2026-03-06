// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package fluxmq

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	authv1 "github.com/absmach/fluxmq/pkg/proto/auth/v1"
	"github.com/absmach/fluxmq/pkg/proto/auth/v1/authv1connect"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	grpcDomainsV1 "github.com/absmach/supermq/api/grpc/domains/v1"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/gofrs/uuid/v5"
)

const (
	defaultClientType = "client"
	publishConnType   = uint32(1)
	subscribeConnType = uint32(2)

	smqMessagesPrefix    = "m"
	smqChannelsSeparator = "c"

	defaultRouteCacheTTL        = 1 * time.Minute
	defaultRouteCacheMaxEntries = 10000
)

var _ authv1connect.AuthServiceHandler = (*AuthServer)(nil)

// AuthServer implements FluxMQ's AuthServiceHandler by delegating to
// SuperMQ's internal gRPC services (Clients, Channels, Domains).
type AuthServer struct {
	clients    grpcClientsV1.ClientsServiceClient
	channels   grpcChannelsV1.ChannelsServiceClient
	domains    grpcDomainsV1.DomainsServiceClient
	clientType string
	logger     *slog.Logger
	routeCache *routeResolutionCache
}

// AuthServerConfig holds configuration for the AuthServer.
type AuthServerConfig struct {
	Clients    grpcClientsV1.ClientsServiceClient
	Channels   grpcChannelsV1.ChannelsServiceClient
	Domains    grpcDomainsV1.DomainsServiceClient
	ClientType string
	Logger     *slog.Logger
}

// NewAuthServer creates a new AuthServer.
func NewAuthServer(cfg AuthServerConfig) *AuthServer {
	clientType := strings.TrimSpace(cfg.ClientType)
	if clientType == "" {
		clientType = defaultClientType
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &AuthServer{
		clients:    cfg.Clients,
		channels:   cfg.Channels,
		domains:    cfg.Domains,
		clientType: clientType,
		logger:     logger,
		routeCache: newRouteResolutionCache(defaultRouteCacheTTL, defaultRouteCacheMaxEntries),
	}
}

// Authenticate validates client credentials via SuperMQ's Clients service.
func (s *AuthServer) Authenticate(
	ctx context.Context,
	req *connect.Request[authv1.AuthnReq],
) (*connect.Response[authv1.AuthnRes], error) {
	msg := req.Msg
	token := authn.AuthPack(authn.BasicAuth, msg.GetUsername(), msg.GetPassword())

	res, err := s.clients.Authenticate(ctx, &grpcClientsV1.AuthnReq{Token: token})
	if err != nil {
		s.logger.Warn("authn_rpc_failed",
			slog.String("client_id", msg.GetClientId()),
			slog.String("error", err.Error()))
		return connect.NewResponse(&authv1.AuthnRes{
			Authenticated: false,
			ReasonCode:    1,
			Reason:        "authentication service error",
		}), nil
	}

	if !res.GetAuthenticated() {
		s.logger.Debug("authn_denied", slog.String("client_id", msg.GetClientId()))
		return connect.NewResponse(&authv1.AuthnRes{
			Authenticated: false,
			ReasonCode:    2,
			Reason:        "bad credentials",
		}), nil
	}

	return connect.NewResponse(&authv1.AuthnRes{
		Authenticated: true,
		Id:            res.GetId(),
	}), nil
}

// Authorize checks topic-level permissions via SuperMQ's Channels service.
// The topic is parsed for the SuperMQ format (m/<domain>/c/<channel>/...).
// Non-SuperMQ topics are allowed by default.
func (s *AuthServer) Authorize(
	ctx context.Context,
	req *connect.Request[authv1.AuthzReq],
) (*connect.Response[authv1.AuthzRes], error) {
	msg := req.Msg
	topic := msg.GetTopic()
	externalID := msg.GetExternalId()

	domainID, channelID, handled, err := parseSuperMQTopic(topic)
	if err != nil {
		s.logger.Warn("authz_invalid_topic",
			slog.String("external_id", externalID),
			slog.String("topic", topic),
			slog.String("error", err.Error()))
		return connect.NewResponse(&authv1.AuthzRes{
			Authorized: false,
			ReasonCode: 1,
			Reason:     err.Error(),
		}), nil
	}
	if !handled {
		return connect.NewResponse(&authv1.AuthzRes{Authorized: true}), nil
	}

	domainID, channelID, err = s.resolveTopicIDs(ctx, domainID, channelID)
	if err != nil {
		s.logger.Warn("authz_resolve_failed",
			slog.String("external_id", externalID),
			slog.String("topic", topic),
			slog.String("error", err.Error()))
		return connect.NewResponse(&authv1.AuthzRes{
			Authorized: false,
			ReasonCode: 2,
			Reason:     "route resolution failed",
		}), nil
	}

	connType := publishConnType
	if msg.GetAction() == authv1.Action_ACTION_SUBSCRIBE {
		connType = subscribeConnType
	}

	authzRes, err := s.channels.Authorize(ctx, &grpcChannelsV1.AuthzReq{
		DomainId:   domainID,
		ClientId:   externalID,
		ClientType: s.clientType,
		ChannelId:  channelID,
		Type:       connType,
	})
	if err != nil {
		s.logger.Warn("authz_rpc_failed",
			slog.String("external_id", externalID),
			slog.String("domain_id", domainID),
			slog.String("channel_id", channelID),
			slog.String("error", err.Error()))
		return connect.NewResponse(&authv1.AuthzRes{
			Authorized: false,
			ReasonCode: 3,
			Reason:     "authorization service error",
		}), nil
	}

	if !authzRes.GetAuthorized() {
		s.logger.Debug("authz_denied",
			slog.String("external_id", externalID),
			slog.String("domain_id", domainID),
			slog.String("channel_id", channelID),
			slog.Uint64("conn_type", uint64(connType)))
		return connect.NewResponse(&authv1.AuthzRes{
			Authorized: false,
			ReasonCode: 4,
			Reason:     "not authorized",
		}), nil
	}

	return connect.NewResponse(&authv1.AuthzRes{Authorized: true}), nil
}

func (s *AuthServer) resolveTopicIDs(ctx context.Context, domainID, channelID string) (string, string, error) {
	if isUUID(domainID) && isUUID(channelID) {
		return domainID, channelID, nil
	}

	cacheKey := routeResolutionCacheKey(domainID, channelID)
	if s.routeCache != nil {
		if cachedDomainID, cachedChannelID, ok := s.routeCache.get(cacheKey); ok {
			return cachedDomainID, cachedChannelID, nil
		}
	}

	if !isUUID(domainID) {
		if s.domains == nil {
			return "", "", fmt.Errorf("domains service client is not configured")
		}
		res, err := s.domains.RetrieveIDByRoute(ctx, &grpcCommonV1.RetrieveIDByRouteReq{
			Route: domainID,
		})
		if err != nil {
			return "", "", err
		}
		domainID = res.GetEntity().GetId()
	}

	if !isUUID(channelID) {
		if s.channels == nil {
			return "", "", fmt.Errorf("channels service client is not configured")
		}
		res, err := s.channels.RetrieveIDByRoute(ctx, &grpcCommonV1.RetrieveIDByRouteReq{
			Route:    channelID,
			DomainId: domainID,
		})
		if err != nil {
			return "", "", err
		}
		channelID = res.GetEntity().GetId()
	}

	if s.routeCache != nil {
		s.routeCache.set(cacheKey, domainID, channelID)
	}

	return domainID, channelID, nil
}

// Shared helpers (same as in authorizer.go but needed for the server).

func isUUID(id string) bool {
	parsed, err := uuid.FromString(id)
	return err == nil && parsed.String() == id
}

type routeResolutionCacheEntry struct {
	domainID  string
	channelID string
	expiresAt time.Time
}

type routeResolutionCache struct {
	mu         sync.RWMutex
	items      map[string]routeResolutionCacheEntry
	ttl        time.Duration
	maxEntries int
}

func newRouteResolutionCache(ttl time.Duration, maxEntries int) *routeResolutionCache {
	if ttl <= 0 {
		ttl = defaultRouteCacheTTL
	}
	if maxEntries <= 0 {
		maxEntries = defaultRouteCacheMaxEntries
	}
	return &routeResolutionCache{
		items:      make(map[string]routeResolutionCacheEntry, maxEntries),
		ttl:        ttl,
		maxEntries: maxEntries,
	}
}

func routeResolutionCacheKey(domainID, channelID string) string {
	return domainID + "|" + channelID
}

func (c *routeResolutionCache) get(key string) (domainID, channelID string, ok bool) {
	now := time.Now()
	c.mu.RLock()
	entry, found := c.items[key]
	c.mu.RUnlock()
	if !found {
		return "", "", false
	}
	if now.After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return "", "", false
	}
	return entry.domainID, entry.channelID, true
}

func (c *routeResolutionCache) set(key, domainID, channelID string) {
	now := time.Now()
	c.mu.Lock()
	if len(c.items) >= c.maxEntries {
		for k, entry := range c.items {
			if now.After(entry.expiresAt) {
				delete(c.items, k)
			}
		}
		if len(c.items) >= c.maxEntries {
			for k := range c.items {
				delete(c.items, k)
				break
			}
		}
	}
	c.items[key] = routeResolutionCacheEntry{
		domainID:  domainID,
		channelID: channelID,
		expiresAt: now.Add(c.ttl),
	}
	c.mu.Unlock()
}

func parseSuperMQTopic(topic string) (domainID, channelID string, handled bool, err error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(topic), "/")
	if trimmed == "" {
		return "", "", false, nil
	}
	parts := strings.Split(trimmed, "/")
	if parts[0] != smqMessagesPrefix {
		return "", "", false, nil
	}
	if len(parts) < 4 {
		return "", "", true, fmt.Errorf("malformed supermq topic: expected m/<domain>/c/<channel>")
	}
	if parts[2] != smqChannelsSeparator {
		return "", "", true, fmt.Errorf("malformed supermq topic: expected m/<domain>/c/<channel>")
	}
	if parts[1] == "" || parts[3] == "" {
		return "", "", true, fmt.Errorf("malformed supermq topic: empty domain or channel")
	}
	if strings.ContainsAny(parts[1], "+#") || strings.ContainsAny(parts[3], "+#") {
		return "", "", true, fmt.Errorf("malformed supermq topic: wildcards not allowed in domain/channel")
	}
	return parts[1], parts[3], true, nil
}
