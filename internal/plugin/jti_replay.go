package plugin

import (
	"sync"
	"time"
)

// JTIReplayCache tracks JWT IDs (jti) to prevent token replay attacks.
// It stores seen JTIs with per-entry TTLs based on the token's remaining
// lifetime. Entries are automatically evicted on access.
type JTIReplayCache struct {
	mu      sync.Mutex
	entries map[string]time.Time // jti -> expiry
	now     func() time.Time
}

// NewJTIReplayCache creates a replay cache with periodic cleanup.
func NewJTIReplayCache() *JTIReplayCache {
	c := &JTIReplayCache{
		entries: make(map[string]time.Time),
		now:     time.Now,
	}
	go c.cleanupLoop(5 * time.Minute)
	return c
}

// Seen returns true if the jti has been seen and is not yet expired.
func (c *JTIReplayCache) Seen(jti string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	exp, ok := c.entries[jti]
	if !ok {
		return false
	}
	if c.now().After(exp) {
		delete(c.entries, jti)
		return false
	}
	return true
}

// Add registers a jti with the given TTL.
func (c *JTIReplayCache) Add(jti string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[jti] = c.now().Add(ttl)
}

// Len returns the current number of entries (for testing).
func (c *JTIReplayCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.evictExpired()
	return len(c.entries)
}

func (c *JTIReplayCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		c.evictExpired()
		c.mu.Unlock()
	}
}

func (c *JTIReplayCache) evictExpired() {
	now := c.now()
	for jti, exp := range c.entries {
		if now.After(exp) {
			delete(c.entries, jti)
		}
	}
}
