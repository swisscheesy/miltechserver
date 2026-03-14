package ps_mag

import (
	"sync"
	"time"
)

// issueCache is a single-entry TTL cache for the full ps-mag issue list.
// Thread-safe for concurrent access.
type issueCache struct {
	mu        sync.RWMutex
	issues    []PSMagIssueResponse
	expiresAt time.Time
	ttl       time.Duration
}

func newIssueCache(ttl time.Duration) *issueCache {
	return &issueCache{ttl: ttl}
}

// get returns a copy of the cached issue list and true if the cache is warm and
// not expired. Returns nil and false on a cache miss.
func (c *issueCache) get() ([]PSMagIssueResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.issues == nil || time.Now().After(c.expiresAt) {
		return nil, false
	}
	// Return a copy so callers cannot mutate the cached slice.
	cp := make([]PSMagIssueResponse, len(c.issues))
	copy(cp, c.issues)
	return cp, true
}

// set stores a defensive copy of issues in the cache and resets the expiry clock.
// The copy is made before acquiring the lock to keep the critical section minimal.
func (c *issueCache) set(issues []PSMagIssueResponse) {
	cp := make([]PSMagIssueResponse, len(issues))
	copy(cp, issues)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.issues = cp
	c.expiresAt = time.Now().Add(c.ttl)
}
