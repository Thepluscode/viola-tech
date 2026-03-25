package ratelimit

import (
	"sync"
	"time"
)

// Limiter implements a sliding window rate limiter
type Limiter struct {
	mu sync.RWMutex

	// Per-key sliding windows
	windows map[string]*slidingWindow

	// Configuration
	windowSize time.Duration
	maxRequests int

	// Cleanup
	lastCleanup time.Time
	cleanupInterval time.Duration
}

type slidingWindow struct {
	requests []time.Time
}

// Config configures the rate limiter
type Config struct {
	WindowSize time.Duration // e.g., 1 minute
	MaxRequests int          // e.g., 120 requests
	CleanupInterval time.Duration // e.g., 5 minutes
}

// New creates a new rate limiter
func New(cfg Config) *Limiter {
	if cfg.WindowSize == 0 {
		cfg.WindowSize = 1 * time.Minute
	}
	if cfg.MaxRequests == 0 {
		cfg.MaxRequests = 120
	}
	if cfg.CleanupInterval == 0 {
		cfg.CleanupInterval = 5 * time.Minute
	}

	return &Limiter{
		windows: make(map[string]*slidingWindow),
		windowSize: cfg.WindowSize,
		maxRequests: cfg.MaxRequests,
		lastCleanup: time.Now(),
		cleanupInterval: cfg.CleanupInterval,
	}
}

// Allow checks if a request should be allowed for the given key
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Periodic cleanup of old windows
	if now.Sub(l.lastCleanup) > l.cleanupInterval {
		l.cleanup(now)
		l.lastCleanup = now
	}

	// Get or create window for this key
	window, exists := l.windows[key]
	if !exists {
		window = &slidingWindow{
			requests: make([]time.Time, 0, l.maxRequests),
		}
		l.windows[key] = window
	}

	// Remove requests outside the sliding window
	cutoff := now.Add(-l.windowSize)
	validRequests := make([]time.Time, 0, len(window.requests))
	for _, reqTime := range window.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	window.requests = validRequests

	// Check if we're within the limit
	if len(window.requests) >= l.maxRequests {
		return false // Rate limit exceeded
	}

	// Record this request
	window.requests = append(window.requests, now)
	return true
}

// Remaining returns how many requests are remaining for the key
func (l *Limiter) Remaining(key string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	window, exists := l.windows[key]
	if !exists {
		return l.maxRequests
	}

	now := time.Now()
	cutoff := now.Add(-l.windowSize)

	count := 0
	for _, reqTime := range window.requests {
		if reqTime.After(cutoff) {
			count++
		}
	}

	remaining := l.maxRequests - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Limit returns the maximum number of requests allowed
func (l *Limiter) Limit() int {
	return l.maxRequests
}

// cleanup removes stale windows to prevent memory leaks
func (l *Limiter) cleanup(now time.Time) {
	cutoff := now.Add(-l.windowSize * 2) // Keep windows for 2x window size

	for key, window := range l.windows {
		// If all requests are old, remove the window
		allOld := true
		for _, reqTime := range window.requests {
			if reqTime.After(cutoff) {
				allOld = false
				break
			}
		}

		if allOld {
			delete(l.windows, key)
		}
	}
}
