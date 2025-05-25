// Package oauth provides OAuth middleware for httpio clients.
//
// This middleware handles OAuth 2.0 authentication by automatically managing access tokens,
// including token acquisition, caching, and refreshing when expired.
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/anggasct/httpio/internal/middleware"
)

// TokenResponse represents an OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// Config represents the OAuth middleware configuration
type Config struct {
	// TokenURL is the URL for token requests
	TokenURL string
	// ClientID is the OAuth client ID
	ClientID string
	// ClientSecret is the OAuth client secret
	ClientSecret string
	// Scopes are the OAuth scopes to request
	Scopes []string
	// GrantType is the OAuth grant type (e.g., "client_credentials", "password", "authorization_code")
	GrantType string
	// Username and Password are used for the "password" grant type
	Username string
	Password string
	// AdditionalParams contains any additional parameters to include in the token request
	AdditionalParams map[string]string
	// HeaderName is the header name for the token (default: "Authorization")
	HeaderName string
	// HeaderFormat is the format for the token header value (default: "Bearer %s")
	HeaderFormat string
	// RefreshThreshold is the time before expiration when the token should be refreshed
	// This prevents using a token that's about to expire
	RefreshThreshold time.Duration
	// OnNewToken is called when a new token is obtained
	OnNewToken func(token *TokenResponse)
	// OnTokenError is called when a token acquisition fails
	OnTokenError func(err error)
}

// DefaultConfig returns a default configuration for the OAuth middleware
func DefaultConfig() *Config {
	return &Config{
		GrantType:        "client_credentials",
		HeaderName:       "Authorization",
		HeaderFormat:     "Bearer %s",
		RefreshThreshold: 30 * time.Second,
	}
}

// Middleware is the OAuth middleware implementation
type Middleware struct {
	config         *Config
	currentToken   *TokenResponse
	tokenExpiresAt time.Time
	mutex          sync.RWMutex
}

// NewMiddleware creates a new OAuth middleware with the provided configuration
func New(config *Config) *Middleware {
	if config == nil {
		config = DefaultConfig()
	}

	if config.HeaderName == "" {
		config.HeaderName = "Authorization"
	}

	if config.HeaderFormat == "" {
		config.HeaderFormat = "Bearer %s"
	}

	return &Middleware{
		config: config,
	}
}

// Handle implements the MiddlewareHandler interface
func (m *Middleware) Handle(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {

		token, err := m.getValidToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("oauth middleware: failed to get token: %w", err)
		}

		req.Header.Set(m.config.HeaderName, fmt.Sprintf(m.config.HeaderFormat, token.AccessToken))

		res, _ := next(ctx, req)
		if res == nil {
			return nil, errors.New("oauth middleware: next handler returned nil response")
		}

		if res != nil && res.StatusCode == http.StatusUnauthorized {
			m.mutex.Lock()
			m.currentToken = nil
			m.mutex.Unlock()
		}

		return res, nil
	}
}

// getValidToken returns a valid token, obtaining a new one if necessary
func (m *Middleware) getValidToken(ctx context.Context) (*TokenResponse, error) {
	m.mutex.RLock()
	if m.currentToken != nil && time.Now().Add(m.config.RefreshThreshold).Before(m.tokenExpiresAt) {
		token := m.currentToken
		m.mutex.RUnlock()
		return token, nil
	}
	m.mutex.RUnlock()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.currentToken != nil && time.Now().Add(m.config.RefreshThreshold).Before(m.tokenExpiresAt) {
		return m.currentToken, nil
	}

	if m.currentToken != nil && m.currentToken.RefreshToken != "" {
		token, err := m.refreshExistingToken(ctx)
		if err == nil {
			m.currentToken = token
			m.tokenExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

			if m.config.OnNewToken != nil {
				m.config.OnNewToken(token)
			}

			return token, nil
		}

		if m.config.OnTokenError != nil {
			m.config.OnTokenError(fmt.Errorf("oauth middleware: refresh token failed, falling back to new token: %w", err))
		}
	}

	token, err := m.fetchNewToken(ctx)
	if err != nil {
		if m.config.OnTokenError != nil {
			m.config.OnTokenError(err)
		}
		return nil, err
	}

	m.currentToken = token
	m.tokenExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	if m.config.OnNewToken != nil {
		m.config.OnNewToken(token)
	}

	return token, nil
}

// refreshExistingToken uses the refresh token to get a new access token
func (m *Middleware) refreshExistingToken(ctx context.Context) (*TokenResponse, error) {
	if m.currentToken == nil || m.currentToken.RefreshToken == "" {
		return nil, fmt.Errorf("oauth middleware: no refresh token available")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", m.currentToken.RefreshToken)
	data.Set("client_id", m.config.ClientID)
	if m.config.ClientSecret != "" {
		data.Set("client_secret", m.config.ClientSecret)
	}

	if len(m.config.Scopes) > 0 {
		data.Set("scope", strings.Join(m.config.Scopes, " "))
	}

	for k, v := range m.config.AdditionalParams {
		data.Set(k, v)
	}

	return m.sendTokenRequest(ctx, data)
}

// fetchNewToken makes an HTTP request to get a new OAuth token
func (m *Middleware) fetchNewToken(ctx context.Context) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", m.config.GrantType)

	switch m.config.GrantType {
	case "client_credentials":
		data.Set("client_id", m.config.ClientID)
		data.Set("client_secret", m.config.ClientSecret)
	case "password":
		data.Set("username", m.config.Username)
		data.Set("password", m.config.Password)
		data.Set("client_id", m.config.ClientID)
		if m.config.ClientSecret != "" {
			data.Set("client_secret", m.config.ClientSecret)
		}
	case "refresh_token":
		if m.currentToken != nil && m.currentToken.RefreshToken != "" {
			data.Set("refresh_token", m.currentToken.RefreshToken)
			data.Set("client_id", m.config.ClientID)
			if m.config.ClientSecret != "" {
				data.Set("client_secret", m.config.ClientSecret)
			}
		} else {
			return nil, fmt.Errorf("oauth middleware: missing refresh token")
		}
	}

	if len(m.config.Scopes) > 0 {
		data.Set("scope", strings.Join(m.config.Scopes, " "))
	}

	for k, v := range m.config.AdditionalParams {
		data.Set(k, v)
	}

	return m.sendTokenRequest(ctx, data)
}

// sendTokenRequest sends a token request to the OAuth server
func (m *Middleware) sendTokenRequest(ctx context.Context, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", m.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth middleware: failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth middleware: token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth middleware: token server returned status %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("oauth middleware: failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}
