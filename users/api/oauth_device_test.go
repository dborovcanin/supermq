// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	"github.com/absmach/supermq/pkg/oauth2"
	"github.com/absmach/supermq/users"
	useroauth "github.com/absmach/supermq/users/oauth"
	"github.com/absmach/supermq/users/oauth/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	goauth2 "golang.org/x/oauth2"
	"google.golang.org/grpc"
)

// These functions are not exported, so we can't test them directly.
// They are tested indirectly through the CreateDeviceCode functionality.

func TestInMemoryDeviceCodeStore(t *testing.T) {
	deviceStore := store.NewInMemoryDeviceCodeStore()

	code := store.DeviceCode{
		DeviceCode:      "device123",
		UserCode:        "ABCD-EFGH",
		VerificationURI: "http://example.com/verify",
		ExpiresIn:       600,
		Interval:        3,
		Provider:        "google",
		CreatedAt:       time.Now(),
		State:           "state123",
	}

	t.Run("Save and Get", func(t *testing.T) {
		err := deviceStore.Save(code)
		assert.NoError(t, err)

		retrieved, err := deviceStore.Get(code.DeviceCode)
		assert.NoError(t, err)
		assert.Equal(t, code.DeviceCode, retrieved.DeviceCode)
		assert.Equal(t, code.UserCode, retrieved.UserCode)
	})

	t.Run("GetByUserCode", func(t *testing.T) {
		retrieved, err := deviceStore.GetByUserCode(code.UserCode)
		assert.NoError(t, err)
		assert.Equal(t, code.DeviceCode, retrieved.DeviceCode)
	})

	t.Run("Update", func(t *testing.T) {
		code.Approved = true
		code.AccessToken = "access_token_123"
		err := deviceStore.Update(code)
		assert.NoError(t, err)

		retrieved, err := deviceStore.Get(code.DeviceCode)
		assert.NoError(t, err)
		assert.True(t, retrieved.Approved)
		assert.Equal(t, "access_token_123", retrieved.AccessToken)
	})

	t.Run("Delete", func(t *testing.T) {
		err := deviceStore.Delete(code.DeviceCode)
		assert.NoError(t, err)

		_, err = deviceStore.Get(code.DeviceCode)
		assert.Error(t, err)

		_, err = deviceStore.GetByUserCode(code.UserCode)
		assert.Error(t, err)
	})

	t.Run("Get non-existent", func(t *testing.T) {
		_, err := deviceStore.Get("nonexistent")
		assert.Error(t, err)
	})

	t.Run("Update non-existent", func(t *testing.T) {
		err := deviceStore.Update(store.DeviceCode{DeviceCode: "nonexistent"})
		assert.Error(t, err)
	})
}

func TestDeviceCodeHandler(t *testing.T) {
	tests := []struct {
		name           string
		providerName   string
		enabled        bool
		setupMocks     func(*MockOAuthService, *MockOAuthProvider)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:         "successful device code generation",
			providerName: "google",
			enabled:      true,
			setupMocks: func(oauthSvc *MockOAuthService, provider *MockOAuthProvider) {
				provider.On("IsEnabled").Return(true)
				mockCode := store.DeviceCode{
					DeviceCode:      "mock-device-code",
					UserCode:        "ABCD-EFGH",
					VerificationURI: "http://example.com/verify",
					ExpiresIn:       600,
					Interval:        5,
				}
				oauthSvc.On("CreateDeviceCode", mock.Anything, provider, mock.Anything).
					Return(mockCode, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var deviceCode store.DeviceCode
				err := json.NewDecoder(rec.Body).Decode(&deviceCode)
				assert.NoError(t, err)
				assert.NotEmpty(t, deviceCode.DeviceCode)
				assert.NotEmpty(t, deviceCode.UserCode)
				assert.NotEmpty(t, deviceCode.VerificationURI)
				assert.Greater(t, deviceCode.ExpiresIn, 0)
				assert.Greater(t, deviceCode.Interval, 0)
			},
		},
		{
			name:         "provider disabled",
			providerName: "google",
			enabled:      false,
			setupMocks: func(oauthSvc *MockOAuthService, provider *MockOAuthProvider) {
				provider.On("IsEnabled").Return(false)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp errorResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "oauth provider is disabled", resp.Error)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider := new(MockOAuthProvider)
			provider.On("Name").Return(tc.providerName)
			oauthSvc := new(MockOAuthService)

			tc.setupMocks(oauthSvc, provider)

			handler := deviceCodeHandler(provider, oauthSvc)

			req := httptest.NewRequest(http.MethodPost, "/oauth/device/code/google", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			tc.checkResponse(t, rec)
		})
	}
}

func TestDeviceTokenHandler(t *testing.T) {
	tests := []struct {
		name           string
		deviceCode     string
		setupMocks     func(*MockOAuthService, *MockOAuthProvider)
		enabled        bool
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "authorization pending",
			deviceCode: "device123",
			setupMocks: func(oauthSvc *MockOAuthService, provider *MockOAuthProvider) {
				provider.On("IsEnabled").Return(true)
				oauthSvc.On("PollDeviceToken", mock.Anything, provider, "device123").
					Return(nil, useroauth.ErrDeviceCodePending)
			},
			enabled:        true,
			expectedStatus: http.StatusAccepted,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp errorResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "authorization pending", resp.Error)
			},
		},
		{
			name:       "invalid device code",
			deviceCode: "invalid",
			setupMocks: func(oauthSvc *MockOAuthService, provider *MockOAuthProvider) {
				provider.On("IsEnabled").Return(true)
				oauthSvc.On("PollDeviceToken", mock.Anything, provider, "invalid").
					Return(nil, useroauth.ErrDeviceCodeNotFound)
			},
			enabled:        true,
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp errorResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Contains(t, resp.Error, "invalid device code")
			},
		},
		{
			name:       "provider disabled",
			deviceCode: "device123",
			setupMocks: func(oauthSvc *MockOAuthService, provider *MockOAuthProvider) {
				provider.On("IsEnabled").Return(false)
			},
			enabled:        false,
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp errorResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "oauth provider is disabled", resp.Error)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider := new(MockOAuthProvider)
			provider.On("Name").Return("google")
			oauthSvc := new(MockOAuthService)

			tc.setupMocks(oauthSvc, provider)

			handler := deviceTokenHandler(provider, oauthSvc)

			reqBody, _ := json.Marshal(map[string]string{
				"device_code": tc.deviceCode,
			})
			req := httptest.NewRequest(http.MethodPost, "/oauth/device/token/google", bytes.NewReader(reqBody))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			tc.checkResponse(t, rec)
		})
	}
}

func TestDeviceVerifyHandler(t *testing.T) {
	tests := []struct {
		name           string
		userCode       string
		code           string
		approve        bool
		setupMocks     func(*MockOAuthService, *MockOAuthProvider)
		enabled        bool
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:     "deny authorization",
			userCode: "ABCD-EFGH",
			code:     "",
			approve:  false,
			setupMocks: func(oauthSvc *MockOAuthService, provider *MockOAuthProvider) {
				oauthSvc.On("GetDeviceCodeByUserCode", mock.Anything, "ABCD-EFGH").
					Return(store.DeviceCode{Provider: "google"}, nil)
				provider.On("IsEnabled").Return(true)
				oauthSvc.On("VerifyDevice", mock.Anything, provider, "ABCD-EFGH", "", false).
					Return(nil)
			},
			enabled:        true,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp map[string]string
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "denied", resp["status"])
			},
		},
		{
			name:     "invalid user code",
			userCode: "INVALID",
			code:     "",
			approve:  false,
			setupMocks: func(oauthSvc *MockOAuthService, provider *MockOAuthProvider) {
				oauthSvc.On("GetDeviceCodeByUserCode", mock.Anything, "INVALID").
					Return(store.DeviceCode{}, useroauth.ErrUserCodeNotFound)
			},
			enabled:        true,
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp errorResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Contains(t, resp.Error, "invalid user code")
			},
		},
		{
			name:     "provider disabled",
			userCode: "ABCD-EFGH",
			code:     "",
			approve:  false,
			setupMocks: func(oauthSvc *MockOAuthService, provider *MockOAuthProvider) {
				oauthSvc.On("GetDeviceCodeByUserCode", mock.Anything, "ABCD-EFGH").
					Return(store.DeviceCode{Provider: "google"}, nil)
				provider.On("IsEnabled").Return(false)
			},
			enabled:        false,
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp errorResponse
				err := json.NewDecoder(rec.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, "oauth provider is disabled", resp.Error)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider := new(MockOAuthProvider)
			provider.On("Name").Return("google")
			oauthSvc := new(MockOAuthService)

			tc.setupMocks(oauthSvc, provider)

			handler := deviceVerifyHandler(oauthSvc, provider)

			reqBody, _ := json.Marshal(map[string]interface{}{
				"user_code": tc.userCode,
				"code":      tc.code,
				"approve":   tc.approve,
			})
			req := httptest.NewRequest(http.MethodPost, "/oauth/device/verify", bytes.NewReader(reqBody))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			tc.checkResponse(t, rec)
		})
	}
}

// Mock OAuth provider for testing
type MockOAuthProvider struct {
	mock.Mock
}

func (m *MockOAuthProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockOAuthProvider) State() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockOAuthProvider) RedirectURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockOAuthProvider) ErrorURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockOAuthProvider) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockOAuthProvider) Exchange(ctx context.Context, code string) (goauth2.Token, error) {
	args := m.Called(ctx, code)
	return args.Get(0).(goauth2.Token), args.Error(1)
}

func (m *MockOAuthProvider) ExchangeWithRedirect(ctx context.Context, code, redirectURL string) (goauth2.Token, error) {
	args := m.Called(ctx, code, redirectURL)
	return args.Get(0).(goauth2.Token), args.Error(1)
}

func (m *MockOAuthProvider) UserInfo(accessToken string) (users.User, error) {
	args := m.Called(accessToken)
	return args.Get(0).(users.User), args.Error(1)
}

func (m *MockOAuthProvider) GetAuthURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockOAuthProvider) GetAuthURLWithRedirect(redirectURL string) string {
	args := m.Called(redirectURL)
	return args.String(0)
}

// Mock OAuth service for testing
type MockOAuthService struct {
	mock.Mock
}

func (m *MockOAuthService) CreateDeviceCode(ctx context.Context, provider oauth2.Provider, verificationURI string) (store.DeviceCode, error) {
	args := m.Called(ctx, provider, verificationURI)
	return args.Get(0).(store.DeviceCode), args.Error(1)
}

func (m *MockOAuthService) PollDeviceToken(ctx context.Context, provider oauth2.Provider, deviceCode string) (*grpcTokenV1.Token, error) {
	args := m.Called(ctx, provider, deviceCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpcTokenV1.Token), args.Error(1)
}

func (m *MockOAuthService) VerifyDevice(ctx context.Context, provider oauth2.Provider, userCode, oauthCode string, approve bool) error {
	args := m.Called(ctx, provider, userCode, oauthCode, approve)
	return args.Error(0)
}

func (m *MockOAuthService) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (store.DeviceCode, error) {
	args := m.Called(ctx, userCode)
	return args.Get(0).(store.DeviceCode), args.Error(1)
}

func (m *MockOAuthService) ProcessWebCallback(ctx context.Context, provider oauth2.Provider, code, state string) (*grpcTokenV1.Token, error) {
	args := m.Called(ctx, provider, code, state)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpcTokenV1.Token), args.Error(1)
}

func (m *MockOAuthService) ProcessDeviceCallback(ctx context.Context, provider oauth2.Provider, userCode, oauthCode string) error {
	args := m.Called(ctx, provider, userCode, oauthCode)
	return args.Error(0)
}

// Mock token service client for testing
type MockTokenServiceClient struct {
	mock.Mock
}

func (m *MockTokenServiceClient) Issue(ctx context.Context, req *grpcTokenV1.IssueReq, opts ...grpc.CallOption) (*grpcTokenV1.Token, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpcTokenV1.Token), args.Error(1)
}

func (m *MockTokenServiceClient) Refresh(ctx context.Context, req *grpcTokenV1.RefreshReq, opts ...grpc.CallOption) (*grpcTokenV1.Token, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpcTokenV1.Token), args.Error(1)
}

// Add these to users/mocks package
func init() {
	// Register mocks to ensure they're available for testing
}
