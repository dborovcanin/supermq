// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans
// to existing traces.
package tracing

import (
	"context"

	"github.com/mainflux/mainflux/auth"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ auth.KeyRepository = (*keyRepositoryMiddleware)(nil)

// keyRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
type keyRepositoryMiddleware struct {
	tracer trace.Tracer
	repo   auth.KeyRepository
}

// New tracks request and their latency, and adds spans
// to context.
func New(repo auth.KeyRepository, tracer trace.Tracer) auth.KeyRepository {
	return keyRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (krm keyRepositoryMiddleware) Save(ctx context.Context, key auth.Key) (string, error) {
	ctx, span := krm.tracer.Start(ctx, "save", trace.WithAttributes(
		attribute.String("id", key.ID),
		attribute.String("type", string(key.Type)),
		attribute.String("subject", key.Subject),
	))
	defer span.End()

	return krm.repo.Save(ctx, key)
}

func (krm keyRepositoryMiddleware) Retrieve(ctx context.Context, owner, id string) (auth.Key, error) {
	ctx, span := krm.tracer.Start(ctx, "retrieve_by_id", trace.WithAttributes(
		attribute.String("id", id),
		attribute.String("owner", owner),
	))
	defer span.End()

	return krm.repo.Retrieve(ctx, owner, id)
}

func (krm keyRepositoryMiddleware) Remove(ctx context.Context, owner, id string) error {
	ctx, span := krm.tracer.Start(ctx, "remove", trace.WithAttributes(
		attribute.String("id", id),
		attribute.String("owner", owner),
	))
	defer span.End()
	return krm.repo.Remove(ctx, owner, id)
}
