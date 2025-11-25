// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/absmach/supermq/pkg/errors"
)

// Token is used for authentication purposes.
// It contains AccessToken, RefreshToken and AccessExpiry.
type Token struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	AccessType   string `json:"access_type,omitempty"`
}

type Login struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (sdk mgSDK) CreateToken(ctx context.Context, lt Login) (Token, errors.SDKError) {
	data, err := json.Marshal(lt)
	if err != nil {
		return Token{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, usersEndpoint, issueTokenEndpoint)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, "", data, nil, http.StatusCreated)
	if sdkErr != nil {
		return Token{}, sdkErr
	}
	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return Token{}, errors.NewSDKError(err)
	}

	return token, nil
}

func (sdk mgSDK) RefreshToken(ctx context.Context, token string) (Token, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, usersEndpoint, refreshTokenEndpoint)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusCreated)
	if sdkErr != nil {
		return Token{}, sdkErr
	}

	t := Token{}
	if err := json.Unmarshal(body, &t); err != nil {
		return Token{}, errors.NewSDKError(err)
	}

	return t, nil
}

// OAuthAuthorizationURL returns the OAuth authorization URL for the given provider.
func (sdk mgSDK) OAuthAuthorizationURL(ctx context.Context, provider, redirectURL string) (string, string, errors.SDKError) {
	reqURL := fmt.Sprintf("%s/oauth/authorize/%s", sdk.usersURL, provider)
	if redirectURL != "" {
		reqURL = fmt.Sprintf("%s?redirect_uri=%s", reqURL, url.QueryEscape(redirectURL))
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, reqURL, "", nil, nil, http.StatusOK)
	if sdkErr != nil {
		return "", "", sdkErr
	}

	var resp struct {
		AuthorizationURL string `json:"authorization_url"`
		State            string `json:"state"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", "", errors.NewSDKError(err)
	}

	return resp.AuthorizationURL, resp.State, nil
}

// OAuthCallback exchanges the OAuth authorization code for tokens.
func (sdk mgSDK) OAuthCallback(ctx context.Context, provider, code, state, redirectURL string) (Token, errors.SDKError) {
	reqURL := fmt.Sprintf("%s/oauth/cli/callback/%s", sdk.usersURL, provider)

	data, err := json.Marshal(map[string]string{
		"code":         code,
		"state":        state,
		"redirect_url": redirectURL,
	})
	if err != nil {
		return Token{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, reqURL, "", data, nil, http.StatusOK)
	if sdkErr != nil {
		return Token{}, sdkErr
	}

	t := Token{}
	if err := json.Unmarshal(body, &t); err != nil {
		return Token{}, errors.NewSDKError(err)
	}

	return t, nil
}
