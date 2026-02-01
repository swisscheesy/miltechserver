package detailed

import (
	"sync"
	"time"

	"miltechserver/api/response"
)

type cacheEntry struct {
	data      response.DetailedResponse
	expiresAt time.Time
}

// Cache provides a TTL-based in-memory cache for detailed item responses.
// Thread-safe for concurrent access.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

// NewCache creates a new cache with the specified TTL in seconds.
// Starts a background cleanup goroutine to evict expired entries.
func NewCache(ttlSeconds int) *Cache {
	c := &Cache{
		entries: make(map[string]cacheEntry),
		ttl:     time.Duration(ttlSeconds) * time.Second,
	}
	go c.cleanup()
	return c
}

// Get retrieves a cached response by NIIN.
// Returns the response and true if found and not expired, otherwise zero value and false.
func (c *Cache) Get(niin string) (response.DetailedResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[niin]
	if !ok || time.Now().After(entry.expiresAt) {
		return response.DetailedResponse{}, false
	}
	return entry.data, true
}

// Set stores a response in the cache with the configured TTL.
func (c *Cache) Set(niin string, data response.DetailedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[niin] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// cleanup runs periodically to remove expired cache entries.
// Runs at half the TTL interval to balance memory usage and CPU overhead.
func (c *Cache) cleanup() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}
