// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	authmocks "github.com/absmach/supermq/auth/mocks"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	oauth2mocks "github.com/absmach/supermq/pkg/oauth2/mocks"
	sdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/absmach/supermq/users"
	httpapi "github.com/absmach/supermq/users/api"
	umocks "github.com/absmach/supermq/users/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	goauth2 "golang.org/x/oauth2"
)

func setupOAuthServer() (*httptest.Server, *umocks.Service, *oauth2mocks.Provider, *authmocks.TokenServiceClient) {
	usvc := new(umocks.Service)
	logger := smqlog.NewMock()
	mux := chi.NewRouter()
	idp := uuid.NewMock()
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return("google")
	authn := new(authnmocks.Authentication)
	am := smqauthn.NewAuthNMiddleware(authn, smqauthn.WithDomainCheck(false), smqauthn.WithAllowUnverifiedUser(true))
	token := new(authmocks.TokenServiceClient)
	httpapi.MakeHandler(usvc, am, token, true, mux, logger, "", passRegex, idp, provider)

	return httptest.NewServer(mux), usvc, provider, token
}

func TestOAuthAuthorizationURL(t *testing.T) {
	ts, _, provider, _ := setupOAuthServer()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		providerName    string
		redirectURL     string
		providerEnabled bool
		getAuthURL      string
		state           string
		err             errors.SDKError
	}{
		{
			desc:            "get authorization URL successfully",
			providerName:    "google",
			redirectURL:     "",
			providerEnabled: true,
			getAuthURL:      "https://accounts.google.com/o/oauth2/auth?client_id=test&state=test-state",
			state:           "test-state",
			err:             nil,
		},
		{
			desc:            "get authorization URL with custom redirect",
			providerName:    "google",
			redirectURL:     "http://localhost:9090/callback",
			providerEnabled: true,
			getAuthURL:      "https://accounts.google.com/o/oauth2/auth?client_id=test&state=test-state&redirect_uri=http://localhost:9090/callback",
			state:           "test-state",
			err:             nil,
		},
		{
			desc:            "get authorization URL with disabled provider",
			providerName:    "google",
			redirectURL:     "",
			providerEnabled: false,
			getAuthURL:      "",
			state:           "",
			err:             errors.NewSDKErrorWithStatus(errors.Wrap(svcerr.ErrNotFound, errors.New("oauth provider is disabled")), http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			provider.On("IsEnabled").Return(tc.providerEnabled).Once()
			if tc.providerEnabled {
				if tc.redirectURL != "" {
					provider.On("GetAuthURLWithRedirect", tc.redirectURL).Return(tc.getAuthURL).Once()
				} else {
					provider.On("GetAuthURL").Return(tc.getAuthURL).Once()
				}
				provider.On("State").Return(tc.state).Once()
			}

			authURL, state, err := mgsdk.OAuthAuthorizationURL(context.Background(), tc.providerName, tc.redirectURL)

			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.getAuthURL, authURL)
				assert.Equal(t, tc.state, state)
			} else {
				assert.Error(t, err)
				assert.Empty(t, authURL)
				assert.Empty(t, state)
			}
		})
	}
}

func TestOAuthCallback(t *testing.T) {
	ts, svc, provider, tokenClient := setupOAuthServer()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	validUser := users.User{
		ID:    generateUUID(t),
		Email: "test@example.com",
		Credentials: users.Credentials{
			Username: "testuser",
		},
		Status: users.EnabledStatus,
	}

	cases := []struct {
		desc            string
		providerName    string
		code            string
		state           string
		redirectURL     string
		providerEnabled bool
		mockSetup       func()
		expectedToken   sdk.Token
		err             errors.SDKError
	}{
		{
			desc:            "successful OAuth callback",
			providerName:    "google",
			code:            "test-code",
			state:           "test-state",
			redirectURL:     "http://localhost:9090/callback",
			providerEnabled: true,
			mockSetup: func() {
				provider.On("IsEnabled").Return(true).Once()
				provider.On("State").Return("test-state").Once()
				provider.On("ExchangeWithRedirect", mock.Anything, "test-code", "http://localhost:9090/callback").
					Return(goauth2.Token{AccessToken: "access-token"}, nil).Once()
				provider.On("UserInfo", "access-token").Return(validUser, nil).Once()
				svc.On("OAuthCallback", mock.Anything, mock.MatchedBy(func(u users.User) bool {
					return u.Email == validUser.Email && u.AuthProvider == "google"
				})).Return(validUser, nil).Once()
				svc.On("OAuthAddUserPolicy", mock.Anything, validUser).Return(nil).Once()
				refreshToken := "jwt-refresh-token"
				tokenClient.On("Issue", mock.Anything, mock.Anything).
					Return(&grpcTokenV1.Token{
						AccessToken:  "jwt-access-token",
						RefreshToken: &refreshToken,
					}, nil).Once()
			},
			expectedToken: sdk.Token{
				AccessToken:  "jwt-access-token",
				RefreshToken: "jwt-refresh-token",
			},
			err: nil,
		},
		{
			desc:            "OAuth callback without redirect URL",
			providerName:    "google",
			code:            "test-code",
			state:           "test-state",
			redirectURL:     "",
			providerEnabled: true,
			mockSetup: func() {
				provider.On("IsEnabled").Return(true).Once()
				provider.On("State").Return("test-state").Once()
				provider.On("Exchange", mock.Anything, "test-code").
					Return(goauth2.Token{AccessToken: "access-token"}, nil).Once()
				provider.On("UserInfo", "access-token").Return(validUser, nil).Once()
				svc.On("OAuthCallback", mock.Anything, mock.MatchedBy(func(u users.User) bool {
					return u.Email == validUser.Email && u.AuthProvider == "google"
				})).Return(validUser, nil).Once()
				svc.On("OAuthAddUserPolicy", mock.Anything, validUser).Return(nil).Once()
				refreshToken := "jwt-refresh-token"
				tokenClient.On("Issue", mock.Anything, mock.Anything).
					Return(&grpcTokenV1.Token{
						AccessToken:  "jwt-access-token",
						RefreshToken: &refreshToken,
					}, nil).Once()
			},
			expectedToken: sdk.Token{
				AccessToken:  "jwt-access-token",
				RefreshToken: "jwt-refresh-token",
			},
			err: nil,
		},
		{
			desc:            "OAuth callback with disabled provider",
			providerName:    "google",
			code:            "test-code",
			state:           "test-state",
			redirectURL:     "",
			providerEnabled: false,
			mockSetup: func() {
				provider.On("IsEnabled").Return(false).Once()
			},
			expectedToken: sdk.Token{},
			err:           errors.NewSDKErrorWithStatus(errors.Wrap(svcerr.ErrNotFound, errors.New("oauth provider is disabled")), http.StatusNotFound),
		},
		{
			desc:            "OAuth callback with invalid state",
			providerName:    "google",
			code:            "test-code",
			state:           "wrong-state",
			redirectURL:     "",
			providerEnabled: true,
			mockSetup: func() {
				provider.On("IsEnabled").Return(true).Once()
				provider.On("State").Return("test-state").Once()
			},
			expectedToken: sdk.Token{},
			err:           errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrMalformedEntity, errors.New("invalid state")), http.StatusBadRequest),
		},
		{
			desc:            "OAuth callback with exchange error",
			providerName:    "google",
			code:            "test-code",
			state:           "test-state",
			redirectURL:     "",
			providerEnabled: true,
			mockSetup: func() {
				provider.On("IsEnabled").Return(true).Once()
				provider.On("State").Return("test-state").Once()
				provider.On("Exchange", mock.Anything, "test-code").
					Return(goauth2.Token{}, fmt.Errorf("exchange failed")).Once()
			},
			expectedToken: sdk.Token{},
			err:           errors.NewSDKErrorWithStatus(errors.New("exchange failed"), http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.mockSetup()

			token, err := mgsdk.OAuthCallback(context.Background(), tc.providerName, tc.code, tc.state, tc.redirectURL)

			if tc.err == nil {
				assert.NoError(t, err)
				assert.NotEmpty(t, token.AccessToken)
				assert.NotEmpty(t, token.RefreshToken)
			} else {
				assert.Error(t, err)
				assert.Empty(t, token.AccessToken)
				assert.Empty(t, token.RefreshToken)
			}

			// Reset mocks
			svc.ExpectedCalls = nil
		})
	}
}

func TestOAuthIntegration(t *testing.T) {
	ts, svc, provider, tokenClient := setupOAuthServer()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	validUser := users.User{
		ID:    generateUUID(t),
		Email: "test@example.com",
		Credentials: users.Credentials{
			Username: "testuser",
		},
		Status: users.EnabledStatus,
	}

	redirectURL := "http://localhost:9090/callback"
	testState := "test-state"
	testCode := "test-code"

	// Setup mocks for authorization URL
	provider.On("IsEnabled").Return(true).Once()
	provider.On("GetAuthURLWithRedirect", redirectURL).
		Return("https://accounts.google.com/o/oauth2/auth?redirect_uri=" + redirectURL).Once()
	provider.On("State").Return(testState).Once()

	// Step 1: Get authorization URL
	authURL, state, err := mgsdk.OAuthAuthorizationURL(context.Background(), "google", redirectURL)
	assert.NoError(t, err)
	assert.NotEmpty(t, authURL)
	assert.Equal(t, testState, state)
	assert.Contains(t, authURL, redirectURL)

	// Setup mocks for callback
	provider.On("IsEnabled").Return(true).Once()
	provider.On("State").Return(testState).Once()
	provider.On("ExchangeWithRedirect", mock.Anything, testCode, redirectURL).
		Return(goauth2.Token{AccessToken: "access-token"}, nil).Once()
	provider.On("UserInfo", "access-token").Return(validUser, nil).Once()
	svc.On("OAuthCallback", mock.Anything, mock.MatchedBy(func(u users.User) bool {
		return u.Email == validUser.Email && u.AuthProvider == "google"
	})).Return(validUser, nil).Once()
	svc.On("OAuthAddUserPolicy", mock.Anything, validUser).Return(nil).Once()
	refreshToken := "jwt-refresh-token"
	tokenClient.On("Issue", mock.Anything, mock.Anything).
		Return(&grpcTokenV1.Token{
			AccessToken:  "jwt-access-token",
			RefreshToken: &refreshToken,
		}, nil).Once()

	// Step 2: Exchange code for token
	token, err := mgsdk.OAuthCallback(context.Background(), "google", testCode, state, redirectURL)
	assert.NoError(t, err)
	assert.NotEmpty(t, token.AccessToken)
	assert.NotEmpty(t, token.RefreshToken)
}
