// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains users main function to start the users service.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"time"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	authSvc "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal"
	jaegerclient "github.com/absmach/magistrala/internal/clients/jaeger"
	pgclient "github.com/absmach/magistrala/internal/clients/postgres"
	"github.com/absmach/magistrala/internal/email"
	mggroups "github.com/absmach/magistrala/internal/groups"
	gapi "github.com/absmach/magistrala/internal/groups/api"
	gevents "github.com/absmach/magistrala/internal/groups/events"
	gpostgres "github.com/absmach/magistrala/internal/groups/postgres"
	gtracing "github.com/absmach/magistrala/internal/groups/tracing"
	"github.com/absmach/magistrala/internal/postgres"
	"github.com/absmach/magistrala/internal/server"
	httpserver "github.com/absmach/magistrala/internal/server/http"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/oauth2"
	kratosoauth "github.com/absmach/magistrala/pkg/oauth2/kratos"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/users"
	capi "github.com/absmach/magistrala/users/api"
	"github.com/absmach/magistrala/users/emailer"
	uevents "github.com/absmach/magistrala/users/events"
	"github.com/absmach/magistrala/users/hasher"
	"github.com/absmach/magistrala/users/kratos"
	ctracing "github.com/absmach/magistrala/users/tracing"
	"github.com/caarlos0/env/v10"
	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/jmoiron/sqlx"
	ory "github.com/ory/client-go"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName         = "users"
	envPrefixDB     = "MG_USERS_DB_"
	envPrefixHTTP   = "MG_USERS_HTTP_"
	envPrefixAuth   = "MG_AUTH_GRPC_"
	envPrefixKratos = "MG_KRATOS_"
	defDB           = "users"
	defSvcHTTPPort  = "9002"

	defKratosRetryCount   = 10
	defKratosRetryWaitMax = 1 * time.Minute

	streamID = "magistrala.users"
)

type config struct {
	LogLevel           string  `env:"MG_USERS_LOG_LEVEL"           envDefault:"info"`
	AdminEmail         string  `env:"MG_USERS_ADMIN_EMAIL"         envDefault:"admin@example.com"`
	AdminPassword      string  `env:"MG_USERS_ADMIN_PASSWORD"      envDefault:"12345678"`
	PassRegexText      string  `env:"MG_USERS_PASS_REGEX"          envDefault:"^.{8,}$"`
	ResetURL           string  `env:"MG_TOKEN_RESET_ENDPOINT"      envDefault:"/reset-request"`
	JaegerURL          url.URL `env:"MG_JAEGER_URL"                envDefault:"http://localhost:14268/api/traces"`
	SendTelemetry      bool    `env:"MG_SEND_TELEMETRY"            envDefault:"true"`
	InstanceID         string  `env:"MG_USERS_INSTANCE_ID"         envDefault:""`
	ESURL              string  `env:"MG_ES_URL"                    envDefault:"nats://localhost:4222"`
	TraceRatio         float64 `env:"MG_JAEGER_TRACE_RATIO"        envDefault:"1.0"`
	SelfRegister       bool    `env:"MG_USERS_ALLOW_SELF_REGISTER" envDefault:"false"`
	OAuthUIRedirectURL string  `env:"MG_OAUTH_UI_REDIRECT_URL"     envDefault:"http://localhost:9095/domains"`
	OAuthUIErrorURL    string  `env:"MG_OAUTH_UI_ERROR_URL"        envDefault:"http://localhost:9095/error"`
	KratosURL          string  `env:"MG_KRATOS_URL"                envDefault:"http://localhost:4433"`
	KratosAPIKey       string  `env:"MG_KRATOS_API_KEY"            envDefault:""`
	KratosSchemaID     string  `env:"MG_KRATOS_SCHEMA_ID"          envDefault:""`
	PassRegex          *regexp.Regexp
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}
	passRegex, err := regexp.Compile(cfg.PassRegexText)
	if err != nil {
		log.Fatalf("invalid password validation rules %s\n", cfg.PassRegexText)
	}
	cfg.PassRegex = passRegex

	logger, err := mglog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err.Error())
	}

	var exitCode int
	defer mglog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	ec := email.Config{}
	if err := env.Parse(&ec); err != nil {
		logger.Error(fmt.Sprintf("failed to load email configuration : %s", err.Error()))
		exitCode = 1
		return
	}

	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	gm := gpostgres.Migration()
	db, err := pgclient.Setup(dbConfig, *gm)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	tp, err := jaegerclient.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init Jaeger: %s", err))
		exitCode = 1
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	authConfig := auth.Config{}
	if err := env.ParseWithOptions(&authConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	authClient, authHandler, err := auth.Setup(authConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	csvc, gsvc, err := newService(ctx, authClient, db, dbConfig, tracer, cfg, ec, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup service: %s", err))
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}

	oauthConfig := oauth2.Config{}
	if err := env.ParseWithOptions(&oauthConfig, env.Options{Prefix: envPrefixKratos}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s Kratos configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}
	oauthProvider := kratosoauth.NewProvider(oauthConfig, cfg.KratosURL, cfg.OAuthUIRedirectURL, cfg.OAuthUIErrorURL, cfg.KratosAPIKey)

	mux := chi.NewRouter()
	httpSrv := httpserver.New(ctx, cancel, svcName, httpServerConfig, capi.MakeHandler(csvc, gsvc, mux, logger, cfg.InstanceID, cfg.PassRegex, oauthProvider), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return httpSrv.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, httpSrv)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("users service terminated: %s", err))
	}
}

func newService(ctx context.Context, authClient magistrala.AuthServiceClient, db *sqlx.DB, dbConfig pgclient.Config, tracer trace.Tracer, c config, ec email.Config, logger *slog.Logger) (users.Service, groups.Service, error) {
	hsr := hasher.New()

	database := postgres.NewDatabase(db, dbConfig, tracer)

	conf := ory.NewConfiguration()
	conf.Servers = []ory.ServerConfiguration{{URL: c.KratosURL}}
	conf.AddDefaultHeader("Authorization", "Bearer "+c.KratosAPIKey)

	retryClient := retryablehttp.NewClient()
	retryClient.Logger = logger
	retryClient.RetryMax = defKratosRetryCount
	retryClient.RetryWaitMax = defKratosRetryWaitMax
	conf.HTTPClient = retryClient.StandardClient()

	client := ory.NewAPIClient(conf)
	cRepo := kratos.NewRepository(client, c.KratosSchemaID, hsr)
	gRepo := gpostgres.New(database)

	idp := uuid.New()

	emailerClient, err := emailer.New(c.ResetURL, &ec)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to configure e-mailing util: %s", err.Error()))
	}

	csvc := users.NewService(cRepo, authClient, emailerClient, c.SelfRegister, client)
	gsvc := mggroups.NewService(gRepo, idp, authClient)

	csvc, err = uevents.NewEventStoreMiddleware(ctx, csvc, c.ESURL)
	if err != nil {
		return nil, nil, err
	}
	gsvc, err = gevents.NewEventStoreMiddleware(ctx, gsvc, c.ESURL, streamID)
	if err != nil {
		return nil, nil, err
	}

	csvc = ctracing.New(csvc, tracer)
	csvc = capi.LoggingMiddleware(csvc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	csvc = capi.MetricsMiddleware(csvc, counter, latency)

	gsvc = gtracing.New(gsvc, tracer)
	gsvc = gapi.LoggingMiddleware(gsvc, logger)
	counter, latency = internal.MakeMetrics("groups", "api")
	gsvc = gapi.MetricsMiddleware(gsvc, counter, latency)

	clientID, err := createAdmin(ctx, c, cRepo, csvc)
	if err != nil {
		return nil, nil, err
	}
	if err := createAdminPolicy(ctx, clientID, authClient); err != nil {
		return nil, nil, err
	}
	return csvc, gsvc, err
}

func createAdmin(ctx context.Context, c config, crepo users.Repository, svc users.Service) (string, error) {
	client := mgclients.Client{
		Name: "admin-client",
		Credentials: mgclients.Credentials{
			Identity: c.AdminEmail,
			Secret:   c.AdminPassword,
		},
		Metadata: mgclients.Metadata{
			"role": "admin",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Role:      mgclients.AdminRole,
		Status:    mgclients.EnabledStatus,
	}

	if c, err := crepo.RetrieveByIdentity(ctx, client.Credentials.Identity); err == nil {
		return c.ID, nil
	}

	// Create an admin
	if _, err := crepo.Save(ctx, client); err != nil {
		return "", err
	}
	if _, err := svc.IssueToken(ctx, c.AdminEmail, c.AdminPassword, ""); err != nil {
		return "", err
	}
	return client.ID, nil
}

func createAdminPolicy(ctx context.Context, clientID string, authClient magistrala.AuthServiceClient) error {
	res, err := authClient.Authorize(ctx, &magistrala.AuthorizeReq{
		SubjectType: authSvc.UserType,
		Subject:     clientID,
		Permission:  authSvc.AdministratorRelation,
		Object:      authSvc.MagistralaObject,
		ObjectType:  authSvc.PlatformType,
	})
	if err != nil || !res.Authorized {
		addPolicyRes, err := authClient.AddPolicy(ctx, &magistrala.AddPolicyReq{
			SubjectType: authSvc.UserType,
			Subject:     clientID,
			Relation:    authSvc.AdministratorRelation,
			Object:      authSvc.MagistralaObject,
			ObjectType:  authSvc.PlatformType,
		})
		if err != nil {
			return err
		}
		if !addPolicyRes.Added {
			return svcerr.ErrAuthorization
		}
	}
	return nil
}
