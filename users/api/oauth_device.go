// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	"github.com/absmach/supermq/pkg/oauth2"
	"github.com/absmach/supermq/users"
	"github.com/go-chi/chi/v5"
)

const (
	deviceCodeLength      = 8  // Length of user code (e.g., "ABCD-EFGH")
	deviceCodeExpiry      = 10 * time.Minute
	deviceCodePollTimeout = 5 * time.Second
	codeCheckInterval     = 3 * time.Second
)

var (
	errDeviceCodeExpired = newErrorResponse("device code expired")
	errDeviceCodePending = newErrorResponse("authorization pending")
	errSlowDown          = newErrorResponse("slow down")
	errAccessDenied      = newErrorResponse("access denied")
)

// DeviceCode represents an OAuth2 device authorization code.
type DeviceCode struct {
	DeviceCode      string    `json:"device_code"`
	UserCode        string    `json:"user_code"`
	VerificationURI string    `json:"verification_uri"`
	ExpiresIn       int       `json:"expires_in"`
	Interval        int       `json:"interval"`
	Provider        string    `json:"-"`
	CreatedAt       time.Time `json:"-"`
	State           string    `json:"-"`
	AccessToken     string    `json:"-"`
	Approved        bool      `json:"-"`
	Denied          bool      `json:"-"`
	LastPoll        time.Time `json:"-"`
}

// DeviceCodeStore manages device authorization codes.
type DeviceCodeStore interface {
	Save(code DeviceCode) error
	Get(deviceCode string) (DeviceCode, error)
	GetByUserCode(userCode string) (DeviceCode, error)
	Update(code DeviceCode) error
	Delete(deviceCode string) error
}

// inMemoryDeviceCodeStore is an in-memory implementation of DeviceCodeStore.
type inMemoryDeviceCodeStore struct {
	mu          sync.RWMutex
	codes       map[string]DeviceCode
	userCodes   map[string]string // maps user code to device code
	cleanupDone chan struct{}
}

// NewInMemoryDeviceCodeStore creates a new in-memory device code store.
func NewInMemoryDeviceCodeStore() DeviceCodeStore {
	store := &inMemoryDeviceCodeStore{
		codes:       make(map[string]DeviceCode),
		userCodes:   make(map[string]string),
		cleanupDone: make(chan struct{}),
	}
	go store.cleanup()
	return store
}

func (s *inMemoryDeviceCodeStore) Save(code DeviceCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code.DeviceCode] = code
	s.userCodes[code.UserCode] = code.DeviceCode
	return nil
}

func (s *inMemoryDeviceCodeStore) Get(deviceCode string) (DeviceCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	code, ok := s.codes[deviceCode]
	if !ok {
		return DeviceCode{}, fmt.Errorf("device code not found")
	}
	return code, nil
}

func (s *inMemoryDeviceCodeStore) GetByUserCode(userCode string) (DeviceCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	deviceCode, ok := s.userCodes[userCode]
	if !ok {
		return DeviceCode{}, fmt.Errorf("user code not found")
	}
	code, ok := s.codes[deviceCode]
	if !ok {
		return DeviceCode{}, fmt.Errorf("device code not found")
	}
	return code, nil
}

func (s *inMemoryDeviceCodeStore) Update(code DeviceCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.codes[code.DeviceCode]; !ok {
		return fmt.Errorf("device code not found")
	}
	s.codes[code.DeviceCode] = code
	return nil
}

func (s *inMemoryDeviceCodeStore) Delete(deviceCode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if code, ok := s.codes[deviceCode]; ok {
		delete(s.userCodes, code.UserCode)
	}
	delete(s.codes, deviceCode)
	return nil
}

func (s *inMemoryDeviceCodeStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for deviceCode, code := range s.codes {
				if now.Sub(code.CreatedAt) > deviceCodeExpiry {
					delete(s.codes, deviceCode)
					delete(s.userCodes, code.UserCode)
				}
			}
			s.mu.Unlock()
		case <-s.cleanupDone:
			return
		}
	}
}

// generateUserCode generates a human-friendly code like "ABCD-EFGH".
func generateUserCode() (string, error) {
	b := make([]byte, deviceCodeLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	code = strings.ToUpper(code[:deviceCodeLength])
	// Format as XXXX-XXXX
	if len(code) >= 8 {
		code = code[:4] + "-" + code[4:8]
	}
	return code, nil
}

// generateDeviceCode generates a random device code.
func generateDeviceCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// oauthDeviceHandler registers device flow routes for OAuth2 providers.
func oauthDeviceHandler(r *chi.Mux, store DeviceCodeStore, svc users.Service, tokenClient grpcTokenV1.TokenServiceClient, providers ...oauth2.Provider) *chi.Mux {
	for _, provider := range providers {
		r.Post("/oauth/device/code/"+provider.Name(), deviceCodeHandler(provider, store))
		r.Post("/oauth/device/token/"+provider.Name(), deviceTokenHandler(provider, store, svc, tokenClient))
	}
	// Register verify endpoints once (not per provider)
	r.Get("/oauth/device/verify", deviceVerifyPageHandler())
	r.Post("/oauth/device/verify", deviceVerifyHandler(store, svc, tokenClient, providers...))
	return r
}

// deviceCodeHandler initiates the device authorization flow.
func deviceCodeHandler(provider oauth2.Provider, store DeviceCodeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if !provider.IsEnabled() {
			errResp := newErrorResponse("oauth provider is disabled")
			respondWithJSON(w, http.StatusNotFound, errResp)
			return
		}

		userCode, err := generateUserCode()
		if err != nil {
			respondWithJSON(w, http.StatusInternalServerError, newErrorResponse("failed to generate user code"))
			return
		}

		deviceCode, err := generateDeviceCode()
		if err != nil {
			respondWithJSON(w, http.StatusInternalServerError, newErrorResponse("failed to generate device code"))
			return
		}

		// Build verification URI with proper scheme
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		verificationURI := fmt.Sprintf("%s://%s/oauth/device/verify", scheme, r.Host)

		code := DeviceCode{
			DeviceCode:      deviceCode,
			UserCode:        userCode,
			VerificationURI: verificationURI,
			ExpiresIn:       int(deviceCodeExpiry.Seconds()),
			Interval:        int(codeCheckInterval.Seconds()),
			Provider:        provider.Name(),
			CreatedAt:       time.Now(),
			State:           provider.State(),
		}

		if err := store.Save(code); err != nil {
			respondWithJSON(w, http.StatusInternalServerError, newErrorResponse("failed to save device code"))
			return
		}

		respondWithJSON(w, http.StatusOK, code)
	}
}

// deviceTokenHandler polls for device authorization completion.
func deviceTokenHandler(provider oauth2.Provider, store DeviceCodeStore, svc users.Service, tokenClient grpcTokenV1.TokenServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if !provider.IsEnabled() {
			errResp := newErrorResponse("oauth provider is disabled")
			respondWithJSON(w, http.StatusNotFound, errResp)
			return
		}

		var req struct {
			DeviceCode string `json:"device_code"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithJSON(w, http.StatusBadRequest, errInvalidBody)
			return
		}

		code, err := store.Get(req.DeviceCode)
		if err != nil {
			respondWithJSON(w, http.StatusNotFound, newErrorResponse("invalid device code"))
			return
		}

		// Check expiration
		if time.Since(code.CreatedAt) > deviceCodeExpiry {
			store.Delete(req.DeviceCode)
			respondWithJSON(w, http.StatusBadRequest, errDeviceCodeExpired)
			return
		}

		// Check polling rate
		if time.Since(code.LastPoll) < codeCheckInterval {
			respondWithJSON(w, http.StatusBadRequest, errSlowDown)
			return
		}

		// Update last poll time
		code.LastPoll = time.Now()
		store.Update(code)

		// Check if denied
		if code.Denied {
			store.Delete(req.DeviceCode)
			respondWithJSON(w, http.StatusUnauthorized, errAccessDenied)
			return
		}

		// Check if approved
		if !code.Approved || code.AccessToken == "" {
			respondWithJSON(w, http.StatusAccepted, errDeviceCodePending)
			return
		}

		// Process the OAuth user and issue tokens
		jwt, err := processOAuthUser(r.Context(), provider, code.AccessToken, svc, tokenClient)
		if err != nil {
			store.Delete(req.DeviceCode)
			respondWithJSON(w, http.StatusInternalServerError, newErrorResponse(err.Error()))
			return
		}

		store.Delete(req.DeviceCode)
		jwt.AccessType = ""
		respondWithJSON(w, http.StatusOK, jwt)
	}
}

// deviceVerifyPageHandler serves the HTML page for device verification.
func deviceVerifyPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Device Verification - Magistrala</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #073764;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 48px;
            max-width: 500px;
            width: 100%;
            text-align: center;
        }
        h1 {
            color: #073764;
            font-size: 32px;
            margin-bottom: 24px;
        }
        p {
            color: #4a5568;
            font-size: 18px;
            line-height: 1.6;
            margin-bottom: 24px;
        }
        .code-input {
            width: 100%;
            padding: 16px;
            font-size: 24px;
            text-align: center;
            border: 2px solid #e2e8f0;
            border-radius: 8px;
            margin-bottom: 24px;
            text-transform: uppercase;
            letter-spacing: 4px;
        }
        .code-input:focus {
            outline: none;
            border-color: #073764;
        }
        button {
            width: 100%;
            padding: 16px;
            font-size: 18px;
            font-weight: 600;
            color: white;
            background: #073764;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            transition: background 0.2s;
        }
        button:hover {
            background: #05294a;
        }
        button:disabled {
            background: #cbd5e0;
            cursor: not-allowed;
        }
        .message {
            margin-top: 24px;
            padding: 16px;
            border-radius: 8px;
            display: none;
        }
        .message.success {
            background: #d1fae5;
            color: #065f46;
            display: block;
        }
        .message.error {
            background: #fee2e2;
            color: #991b1b;
            display: block;
        }
        @media (max-width: 600px) {
            .container { padding: 32px 24px; }
            h1 { font-size: 28px; }
            .code-input { font-size: 20px; }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Device Verification</h1>
        <p>Enter the code displayed on your device</p>
        <form id="verifyForm">
            <input
                type="text"
                id="userCode"
                class="code-input"
                placeholder="XXXX-XXXX"
                maxlength="9"
                pattern="[A-Za-z0-9]{4}-[A-Za-z0-9]{4}"
                required
            />
            <button type="submit" id="submitBtn">Verify Device</button>
        </form>
        <div id="message" class="message"></div>
    </div>

    <script>
        const form = document.getElementById('verifyForm');
        const input = document.getElementById('userCode');
        const submitBtn = document.getElementById('submitBtn');
        const message = document.getElementById('message');

        // Auto-format input
        input.addEventListener('input', (e) => {
            let value = e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, '');
            if (value.length > 4) {
                value = value.slice(0, 4) + '-' + value.slice(4, 8);
            }
            e.target.value = value;
        });

        form.addEventListener('submit', async (e) => {
            e.preventDefault();

            const userCode = input.value.trim();
            if (!userCode || userCode.length < 9) {
                showMessage('Please enter a valid code', 'error');
                return;
            }

            submitBtn.disabled = true;
            submitBtn.textContent = 'Verifying...';

            try {
                // Get authorization URL for the provider (Google as default)
                // Build the callback URL dynamically from current location
                const callbackURL = window.location.protocol + '//' + window.location.host + '/oauth/callback/google';
                const encodedCallback = encodeURIComponent(callbackURL);

                const authResponse = await fetch('/oauth/authorize/google?redirect_uri=' + encodedCallback);
                if (!authResponse.ok) throw new Error('Failed to get authorization URL');

                const authData = await authResponse.json();

                // Build OAuth URL with device state
                const deviceState = 'device:' + userCode;
                const authURL = authData.authorization_url.replace(/state=[^&]+/, 'state=' + encodeURIComponent(deviceState));

                // Redirect to OAuth provider
                window.location.href = authURL;

            } catch (error) {
                showMessage('Verification failed: ' + error.message, 'error');
                submitBtn.disabled = false;
                submitBtn.textContent = 'Verify Device';
            }
        });

        function showMessage(text, type) {
            message.textContent = text;
            message.className = 'message ' + type;
        }
    </script>
</body>
</html>`
		fmt.Fprint(w, html)
	}
}

// deviceVerifyHandler handles user verification of device codes.
func deviceVerifyHandler(store DeviceCodeStore, svc users.Service, tokenClient grpcTokenV1.TokenServiceClient, providers ...oauth2.Provider) http.HandlerFunc {
	providerMap := make(map[string]oauth2.Provider)
	for _, p := range providers {
		providerMap[p.Name()] = p
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			UserCode string `json:"user_code"`
			Code     string `json:"code"`
			Approve  bool   `json:"approve"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithJSON(w, http.StatusBadRequest, errInvalidBody)
			return
		}

		code, err := store.GetByUserCode(req.UserCode)
		if err != nil {
			respondWithJSON(w, http.StatusNotFound, newErrorResponse("invalid user code"))
			return
		}

		provider, ok := providerMap[code.Provider]
		if !ok {
			respondWithJSON(w, http.StatusBadRequest, newErrorResponse("invalid provider"))
			return
		}

		if !provider.IsEnabled() {
			respondWithJSON(w, http.StatusNotFound, newErrorResponse("oauth provider is disabled"))
			return
		}

		// Check expiration
		if time.Since(code.CreatedAt) > deviceCodeExpiry {
			store.Delete(code.DeviceCode)
			respondWithJSON(w, http.StatusBadRequest, errDeviceCodeExpired)
			return
		}

		if !req.Approve {
			code.Denied = true
			store.Update(code)
			respondWithJSON(w, http.StatusOK, map[string]string{"status": "denied"})
			return
		}

		// Exchange authorization code for access token
		token, err := provider.Exchange(r.Context(), req.Code)
		if err != nil {
			respondWithJSON(w, http.StatusUnauthorized, newErrorResponse(err.Error()))
			return
		}

		code.Approved = true
		code.AccessToken = token.AccessToken
		if err := store.Update(code); err != nil {
			respondWithJSON(w, http.StatusInternalServerError, newErrorResponse("failed to update device code"))
			return
		}

		respondWithJSON(w, http.StatusOK, map[string]string{"status": "approved"})
	}
}
