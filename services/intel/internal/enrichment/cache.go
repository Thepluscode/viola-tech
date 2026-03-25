package enrichment

import (
	"context"
	"sync"
	"time"
)

// CachedClient wraps Client with a simple in-memory TTL cache.
// Repeated lookups for the same indicator within cacheTTL are served from
// memory without hitting the OTX API.
type CachedClient struct {
	inner    *Client
	ttl      time.Duration
	mu       sync.RWMutex
	entries  map[string]cacheEntry
}

type cacheEntry struct {
	result  *Result
	expires time.Time
}

// NewCachedClient wraps a Client with a TTL cache.
// ttl=0 disables caching (every call hits OTX).
func NewCachedClient(inner *Client, ttl time.Duration) *CachedClient {
	return &CachedClient{
		inner:   inner,
		ttl:     ttl,
		entries: make(map[string]cacheEntry),
	}
}

func (c *CachedClient) LookupIP(ctx context.Context, ip string) (*Result, error) {
	return c.cached(ctx, "ip:"+ip, func() (*Result, error) {
		return c.inner.LookupIP(ctx, ip)
	})
}

func (c *CachedClient) LookupDomain(ctx context.Context, domain string) (*Result, error) {
	return c.cached(ctx, "domain:"+domain, func() (*Result, error) {
		return c.inner.LookupDomain(ctx, domain)
	})
}

func (c *CachedClient) LookupHash(ctx context.Context, hash string) (*Result, error) {
	return c.cached(ctx, "hash:"+hash, func() (*Result, error) {
		return c.inner.LookupHash(ctx, hash)
	})
}

func (c *CachedClient) cached(ctx context.Context, key string, fn func() (*Result, error)) (*Result, error) {
	if c.ttl > 0 {
		c.mu.RLock()
		if e, ok := c.entries[key]; ok && time.Now().Before(e.expires) {
			c.mu.RUnlock()
			return e.result, nil
		}
		c.mu.RUnlock()
	}

	result, err := fn()
	if err != nil {
		return nil, err
	}

	if c.ttl > 0 && result != nil {
		c.mu.Lock()
		c.entries[key] = cacheEntry{result: result, expires: time.Now().Add(c.ttl)}
		c.mu.Unlock()
	}
	return result, nil
}

// Purge removes all expired cache entries. Call periodically to avoid memory growth.
func (c *CachedClient) Purge() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, e := range c.entries {
		if now.After(e.expires) {
			delete(c.entries, k)
		}
	}
}
