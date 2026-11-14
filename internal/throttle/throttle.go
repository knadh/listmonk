// Package throttle provides a simple in-memory key store to prevent actions from
// being performed frequently.
package throttle

import (
	"sync"
	"time"
)

// Throttle tracks a set of keys in-memory until they expire at the given TTL.
type Throttle struct {
	mu      sync.Mutex
	ttl     time.Duration
	maxLen  int
	expires map[string]time.Time
	cleanAt time.Time
}

// New creates a throttle store that stores max cap items.
func New(ttl time.Duration, cap int) *Throttle {
	if ttl <= 0 {
		ttl = time.Minute * 10
	}
	if cap <= 0 {
		cap = 5000
	}

	return &Throttle{
		ttl:     ttl,
		maxLen:  cap,
		expires: make(map[string]time.Time),
	}
}

// Insert reserves the given key for the configured TTL. It returns false if
// the key is already throttled or the store is full.
func (c *Throttle) Insert(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clean up expired keys at every ttl.
	now := time.Now()
	if !now.Before(c.cleanAt) {
		for key, expiry := range c.expires {
			if !now.Before(expiry) {
				delete(c.expires, key)
			}
		}

		c.cleanAt = now.Add(min(c.ttl, time.Minute))
	}

	// Key exists and is still within the TTL.
	if expiry, ok := c.expires[key]; ok {
		if now.Before(expiry) {
			return false
		}
	} else if len(c.expires) >= c.maxLen {
		return false
	}
	c.expires[key] = now.Add(c.ttl)

	// New unthrottled key.
	return true
}
