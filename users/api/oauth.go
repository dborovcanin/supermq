package api

import (
	"encoding/json"
	"net/http"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	smqauth "github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/oauth2"
	"github.com/absmach/supermq/users"
	"github.com/go-chi/chi/v5"
	goauth2 "golang.org/x/oauth2"
)

func oauthHandler(r *chi.Mux, svc users.Service, tokenClient grpcTokenV1.TokenServiceClient, providers ...oauth2.Provider) *chi.Mux {
	for _, provider := range providers {
		r.HandleFunc("/oauth/callback/"+provider.Name(), oauth2CallbackHandler(provider, svc, tokenClient))
		r.Get("/oauth/authorize/"+provider.Name(), oauth2AuthorizeHandler(provider))
		r.Post("/oauth/cli/callback/"+provider.Name(), oauth2CLICallbackHandler(provider, svc, tokenClient))
	}

	return r
}

// oauth2CallbackHandler is a http.HandlerFunc that handles OAuth2 callbacks.
func oauth2CallbackHandler(oauth oauth2.Provider, svc users.Service, tokenClient grpcTokenV1.TokenServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !oauth.IsEnabled() {
			http.Redirect(w, r, oauth.ErrorURL()+"?error=oauth%20provider%20is%20disabled", http.StatusSeeOther)
			return
		}
		state := r.FormValue("state")
		if state != oauth.State() {
			http.Redirect(w, r, oauth.ErrorURL()+"?error=invalid%20state", http.StatusSeeOther)
			return
		}

		if code := r.FormValue("code"); code != "" {
			token, err := oauth.Exchange(r.Context(), code)
			if err != nil {
				http.Redirect(w, r, oauth.ErrorURL()+"?error="+err.Error(), http.StatusSeeOther)
				return
			}

			user, err := oauth.UserInfo(token.AccessToken)
			if err != nil {
				http.Redirect(w, r, oauth.ErrorURL()+"?error="+err.Error(), http.StatusSeeOther)
				return
			}

			user.AuthProvider = oauth.Name()
			if user.AuthProvider == "" {
				user.AuthProvider = "oauth"
			}
			user, err = svc.OAuthCallback(r.Context(), user)
			if err != nil {
				http.Redirect(w, r, oauth.ErrorURL()+"?error="+err.Error(), http.StatusSeeOther)
				return
			}
			if err := svc.OAuthAddUserPolicy(r.Context(), user); err != nil {
				http.Redirect(w, r, oauth.ErrorURL()+"?error="+err.Error(), http.StatusSeeOther)
				return
			}

			jwt, err := tokenClient.Issue(r.Context(), &grpcTokenV1.IssueReq{
				UserId:   user.ID,
				Type:     uint32(smqauth.AccessKey),
				UserRole: uint32(smqauth.UserRole),
				Verified: !user.VerifiedAt.IsZero(),
			})
			if err != nil {
				http.Redirect(w, r, oauth.ErrorURL()+"?error="+err.Error(), http.StatusSeeOther)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "access_token",
				Value:    jwt.GetAccessToken(),
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
			})
			http.SetCookie(w, &http.Cookie{
				Name:     "refresh_token",
				Value:    jwt.GetRefreshToken(),
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
			})

			http.Redirect(w, r, oauth.RedirectURL(), http.StatusFound)
			return
		}

		http.Redirect(w, r, oauth.ErrorURL()+"?error=empty%20code", http.StatusSeeOther)
	}
}

// oauth2AuthorizeHandler returns the authorization URL for the OAuth2 provider.
func oauth2AuthorizeHandler(oauth oauth2.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !oauth.IsEnabled() {
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "oauth provider is disabled"}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		redirectURL := r.URL.Query().Get("redirect_uri")
		var authURL string
		if redirectURL != "" {
			authURL = oauth.GetAuthURLWithRedirect(redirectURL)
		} else {
			authURL = oauth.GetAuthURL()
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"authorization_url": authURL,
			"state":             oauth.State(),
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// oauth2CLICallbackHandler handles OAuth2 callbacks for CLI and returns JSON tokens.
func oauth2CLICallbackHandler(oauth oauth2.Provider, svc users.Service, tokenClient grpcTokenV1.TokenServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if !oauth.IsEnabled() {
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "oauth provider is disabled"}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		var req struct {
			Code        string `json:"code"`
			State       string `json:"state"`
			RedirectURL string `json:"redirect_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		if req.State != oauth.State() {
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "invalid state"}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		if req.Code == "" {
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "empty code"}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		var token goauth2.Token
		var err error
		if req.RedirectURL != "" {
			token, err = oauth.ExchangeWithRedirect(r.Context(), req.Code, req.RedirectURL)
		} else {
			token, err = oauth.Exchange(r.Context(), req.Code)
		}
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		user, err := oauth.UserInfo(token.AccessToken)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		user.AuthProvider = oauth.Name()
		if user.AuthProvider == "" {
			user.AuthProvider = "oauth"
		}
		user, err = svc.OAuthCallback(r.Context(), user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
		if err := svc.OAuthAddUserPolicy(r.Context(), user); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		jwt, err := tokenClient.Issue(r.Context(), &grpcTokenV1.IssueReq{
			UserId:   user.ID,
			Type:     uint32(smqauth.AccessKey),
			UserRole: uint32(smqauth.UserRole),
			Verified: !user.VerifiedAt.IsZero(),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		jwt.AccessType = ""
		if err := json.NewEncoder(w).Encode(jwt); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
