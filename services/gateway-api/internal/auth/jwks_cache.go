package auth

import (
	"context"
	"crypto"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"
)

// JWKSPublicKeyProvider provides public keys from JWKS
type JWKSPublicKeyProvider interface {
	Get(ctx context.Context, kid string) (crypto.PublicKey, bool)
	Refresh(ctx context.Context) error
}

// JWKSCache caches JWKS keys with automatic refresh.
// Call StartBackground to begin periodic refresh; the goroutine exits when
// the supplied context is cancelled (H4: tied to application lifecycle).
type JWKSCache struct {
	mu sync.RWMutex

	jwksURL string
	http    *http.Client

	keys map[string]crypto.PublicKey

	lastRefresh  time.Time
	refreshEvery time.Duration

	// appCtx is set by StartBackground and used for async refresh requests
	// so the goroutine exits cleanly on application shutdown.
	appCtx context.Context
}

// JWKSCacheConfig configures JWKS caching
type JWKSCacheConfig struct {
	JWKSURL      string
	HTTPClient   *http.Client
	RefreshEvery time.Duration // e.g. 10m
}

// NewJWKSCache creates a new JWKS cache.
// You must call Refresh(ctx) once at startup to populate the key set before
// serving traffic, then optionally call StartBackground to keep it fresh.
func NewJWKSCache(cfg JWKSCacheConfig) (*JWKSCache, error) {
	if cfg.JWKSURL == "" {
		return nil, errors.New("jwks url required")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.RefreshEvery == 0 {
		cfg.RefreshEvery = 10 * time.Minute
	}
	return &JWKSCache{
		jwksURL:      cfg.JWKSURL,
		http:         cfg.HTTPClient,
		keys:         map[string]crypto.PublicKey{},
		refreshEvery: cfg.RefreshEvery,
		appCtx:       context.Background(), // safe default; overwritten by StartBackground
	}, nil
}

// StartBackground launches a goroutine that refreshes the JWKS key set on
// the configured interval. The goroutine exits when ctx is cancelled, making
// it safe to use the application root context for clean shutdown (H4 fix).
func (c *JWKSCache) StartBackground(ctx context.Context) {
	c.mu.Lock()
	c.appCtx = ctx
	c.mu.Unlock()

	go func() {
		ticker := time.NewTicker(c.refreshEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return // Application shutting down — exit cleanly.
			case <-ticker.C:
				if err := c.Refresh(ctx); err != nil {
					log.Printf("WARN jwks refresh: %v", err)
				}
			}
		}
	}()
}

// Get retrieves a public key by kid.
// An on-demand async refresh is triggered when the cache is stale, using the
// application context stored by StartBackground (not context.Background).
func (c *JWKSCache) Get(ctx context.Context, kid string) (crypto.PublicKey, bool) {
	c.maybeRefreshAsync()

	c.mu.RLock()
	defer c.mu.RUnlock()
	k, ok := c.keys[kid]
	return k, ok
}

// Refresh fetches latest JWKS from the endpoint
func (c *JWKSCache) Refresh(ctx context.Context) error {
	keys, err := fetchJWKS(ctx, c.http, c.jwksURL)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Replace whole set. Rotation tolerance is handled by refresh frequency
	// plus JWT exp constraints; keys disappear only when JWKS stops listing them.
	c.keys = keys
	c.lastRefresh = time.Now().UTC()
	return nil
}

// maybeRefreshAsync triggers a background refresh when the cache is stale.
// Uses the application context (set by StartBackground) so the goroutine
// respects application shutdown — not context.Background() (H4 fix).
func (c *JWKSCache) maybeRefreshAsync() {
	c.mu.RLock()
	needs := time.Since(c.lastRefresh) > c.refreshEvery
	appCtx := c.appCtx
	c.mu.RUnlock()

	if !needs {
		return
	}
	go func() {
		if err := c.Refresh(appCtx); err != nil {
			log.Printf("WARN jwks async refresh: %v", err)
		}
	}()
}
