package sources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	l "github.com/redhatinsights/sources-superkey-worker/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// OIDCTokenProvider manages OIDC access tokens using client credentials flow
type OIDCTokenProvider struct {
	config       *clientcredentials.Config
	provider     *oidc.Provider
	token        *oauth2.Token
	mu           sync.RWMutex
	tokenRefresh time.Time
}

var (
	oidcProvider       *OIDCTokenProvider
	oidcProviderOnce   sync.Once
	oidcProviderErr    error
	oidcConfigured     bool
	oidcConfiguredOnce sync.Once
)

// GetOIDCTokenProvider returns a singleton instance of OIDCTokenProvider
func GetOIDCTokenProvider(ctx context.Context) (*OIDCTokenProvider, error) {
	oidcProviderOnce.Do(func() {

		// Check if OIDC is configured
		if conf.OIDCIssuer == "" || conf.OIDCClientID == "" || conf.OIDCClientSecret == "" {
			oidcProviderErr = fmt.Errorf("OIDC not configured: missing issuer, client ID, or client secret")
			return
		}

		// Initialize OIDC provider
		provider, err := oidc.NewProvider(ctx, conf.OIDCIssuer)
		if err != nil {
			oidcProviderErr = fmt.Errorf("failed to initialize OIDC provider: %w", err)
			return
		}

		// Configure OAuth2 client credentials
		oauth2Config := &clientcredentials.Config{
			ClientID:     conf.OIDCClientID,
			ClientSecret: conf.OIDCClientSecret,
			TokenURL:     provider.Endpoint().TokenURL,
			Scopes:       []string{"openid"},
		}

		oidcProvider = &OIDCTokenProvider{
			config:   oauth2Config,
			provider: provider,
		}

		if l.Log != nil {
			l.Log.WithFields(map[string]interface{}{
				"issuer":    conf.OIDCIssuer,
				"client_id": conf.OIDCClientID,
				"token_url": provider.Endpoint().TokenURL,
			}).Info("OIDC token provider initialized")
		}
	})

	return oidcProvider, oidcProviderErr
}

// GetToken retrieves a valid access token, refreshing if necessary
func (o *OIDCTokenProvider) GetToken(ctx context.Context) (string, error) {
	// Try read lock first for cached token
	o.mu.RLock()
	if o.token != nil && o.token.Valid() && time.Now().Before(o.tokenRefresh) {
		token := o.token.AccessToken
		o.mu.RUnlock()
		return token, nil
	}
	o.mu.RUnlock()

	// Need to refresh token - acquire write lock
	o.mu.Lock()
	defer o.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have refreshed)
	if o.token != nil && o.token.Valid() && time.Now().Before(o.tokenRefresh) {
		return o.token.AccessToken, nil
	}

	// Request new token using client credentials flow
	token, err := o.config.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to obtain OIDC token: %w", err)
	}

	o.token = token
	// Refresh token 30 seconds before expiry
	if token.Expiry.IsZero() {
		// If no expiry is set, refresh after 5 minutes
		o.tokenRefresh = time.Now().Add(5 * time.Minute)
	} else {
		o.tokenRefresh = token.Expiry.Add(-30 * time.Second)
	}

	if l.Log != nil {
		l.Log.WithFields(map[string]interface{}{
			"expiry":        token.Expiry,
			"refresh_after": o.tokenRefresh,
		}).Debug("OIDC token obtained")
	}

	return token.AccessToken, nil
}

// IsOIDCConfigured checks if OIDC authentication is configured (cached result)
func IsOIDCConfigured() bool {
	oidcConfiguredOnce.Do(func() {
		oidcConfigured = conf.OIDCIssuer != "" && conf.OIDCClientID != "" && conf.OIDCClientSecret != ""
	})
	return oidcConfigured
}
