package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/mainflux/mainflux/internal/api"
	"github.com/mainflux/mainflux/internal/apiutil"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func policiesHandler(svc things.Service, r *chi.Mux, logger mflog.Logger) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	r.Post("/connect", otelhttp.NewHandler(kithttp.NewServer(
		connectEndpoint(svc),
		decodeConnReq,
		api.EncodeResponse,
		opts...,
	), "create_thing").ServeHTTP)

	r.Post("/disconnect", otelhttp.NewHandler(kithttp.NewServer(
		disconnectEndpoint(svc),
		decodeConnReq,
		api.EncodeResponse,
		opts...,
	), "create_things").ServeHTTP)

	return r
}

func connectEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connReq)
		if err := req.validate(); err != nil {
			return connnRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		if err := svc.Connect(ctx, req.token, req.ThingID, req.ChannelID, req.Permission); err != nil {
			return connnRes{}, err
		}

		return connnRes{}, nil
	}
}

func disconnectEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connReq)
		if err := req.validate(); err != nil {
			return connnRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		if err := svc.Disconnect(ctx, req.token, req.ThingID, req.ChannelID, req.Permission); err != nil {
			return connnRes{}, err
		}

		return connnRes{}, nil
	}
}

func decodeConnReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := connReq{
		token: apiutil.ExtractBearerToken(r),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	return req, nil
}

type connReq struct {
	token      string
	ThingID    string `json:"thing_id,omitempty"`
	ChannelID  string `json:"channel_id,omitempty"`
	Permission string `json:"permission,omitempty"`
}

func (req *connReq) validate() error {
	if req.ThingID == "" || req.ChannelID == "" {
		return errors.ErrCreateEntity
	}
	return nil
}

type connnRes struct{}

func (res *connnRes) Code() int {
	return http.StatusOK
}

func (res *connnRes) Headers() map[string]string {
	return map[string]string{}
}

func (res *connnRes) Empty() bool {
	return false
}
