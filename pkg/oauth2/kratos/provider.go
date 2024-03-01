// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package kratos

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	mfclients "github.com/absmach/magistrala/pkg/clients"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgoauth2 "github.com/absmach/magistrala/pkg/oauth2"
	ory "github.com/ory/client-go"
	"golang.org/x/oauth2"
)

const (
	providerName     = "kratos"
	defTimeout       = 1 * time.Minute
	userInfoEndpoint = "/userinfo?access_token="
	authEndpoint     = "/oauth2/auth"
	TokenEndpoint    = "/oauth2/token"
)

var scopes = []string{
	"email",
	"profile",
	"offline_access",
}

var _ mgoauth2.Provider = (*config)(nil)

type config struct {
	config        *oauth2.Config
	client        *ory.APIClient
	state         string
	baseURL       string
	uiRedirectURL string
	errorURL      string
}

// NewProvider returns a new Google OAuth provider.
func NewProvider(cfg mgoauth2.Config, baseURL, uiRedirectURL, errorURL, apiKey string) mgoauth2.Provider {
	conf := ory.NewConfiguration()
	conf.Servers = []ory.ServerConfiguration{{URL: baseURL}}
	conf.AddDefaultHeader("Authorization", "Bearer "+apiKey)
	client := ory.NewAPIClient(conf)

	return &config{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  baseURL + authEndpoint,
				TokenURL: baseURL + TokenEndpoint,
			},
			RedirectURL: cfg.RedirectURL,
			Scopes:      scopes,
		},
		client:        client,
		baseURL:       baseURL,
		state:         cfg.State,
		uiRedirectURL: uiRedirectURL,
		errorURL:      errorURL,
	}
}

func (cfg *config) Name() string {
	return providerName
}

func (cfg *config) State() string {
	return cfg.state
}

func (cfg *config) RedirectURL() string {
	return cfg.uiRedirectURL
}

func (cfg *config) ErrorURL() string {
	return cfg.errorURL
}

func (cfg *config) IsEnabled() bool {
	return cfg.config.ClientID != "" && cfg.config.ClientSecret != ""
}

func (cfg *config) UserDetails(ctx context.Context, code string) (mfclients.Client, oauth2.Token, error) {
	token, err := cfg.config.Exchange(ctx, code)
	if err != nil {
		return mfclients.Client{}, oauth2.Token{}, err
	}
	if token.RefreshToken == "" {
		return mfclients.Client{}, oauth2.Token{}, svcerr.ErrAuthentication
	}

	resp, err := http.Get(cfg.baseURL + userInfoEndpoint + url.QueryEscape(token.AccessToken))
	if err != nil {
		return mfclients.Client{}, oauth2.Token{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mfclients.Client{}, oauth2.Token{}, svcerr.ErrAuthentication
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return mfclients.Client{}, oauth2.Token{}, err
	}

	var user struct {
		ID    string `json:"sub"`
		Name  string `json:"preferred_username"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(data, &user); err != nil {
		return mfclients.Client{}, oauth2.Token{}, err
	}

	if user.ID == "" || user.Name == "" || user.Email == "" {
		return mfclients.Client{}, oauth2.Token{}, svcerr.ErrAuthentication
	}

	client := mfclients.Client{
		ID:   user.ID,
		Name: user.Name,
		Credentials: mfclients.Credentials{
			Identity: user.Email,
		},
		Metadata: map[string]interface{}{
			"oauth_provider": providerName,
		},
		Status: mfclients.EnabledStatus,
	}

	return client, *token, nil
}

func (cfg *config) Validate(ctx context.Context, token string) error {
	introspectedToken, resp, err := cfg.client.OAuth2API.IntrospectOAuth2Token(ctx).Token(token).Execute()
	if err != nil {
		return decodeError(resp)
	}
	if !introspectedToken.Active {
		return svcerr.ErrAuthentication
	}

	return nil
}

func (cfg *config) Refresh(ctx context.Context, token string) (oauth2.Token, error) {
	payload := strings.NewReader(fmt.Sprintf("grant_type=refresh_token&refresh_token=" + token + "&scope=" + strings.Join(scopes, "%20")))
	client := &http.Client{
		Timeout: defTimeout,
	}
	req, err := http.NewRequest(http.MethodPost, cfg.config.Endpoint.TokenURL, payload)
	if err != nil {
		return oauth2.Token{}, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Basic "+basicAuth(cfg.config.ClientID, cfg.config.ClientSecret))

	res, err := client.Do(req)
	if err != nil {
		return oauth2.Token{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return oauth2.Token{}, svcerr.ErrAuthentication
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return oauth2.Token{}, err
	}
	var tokenData oauth2.Token
	if err := json.Unmarshal(body, &tokenData); err != nil {
		return oauth2.Token{}, err
	}

	return tokenData, nil
}

func basicAuth(id, secret string) string {
	auth := id + ":" + secret
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func decodeError(response *http.Response) error {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	var content struct {
		Error ory.GenericError `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &content); err != nil {
		return fmt.Errorf("error unmarshalling response body: %w", err)
	}

	return fmt.Errorf("error: %s, reason: %s", content.Error.Message, *content.Error.Reason)
}
