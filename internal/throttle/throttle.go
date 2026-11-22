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
	expires map[string]entry
	cleanAt time.Time
}

type entry struct {
	expiry time.Time
	count  int
}

// New creates a throttle store with a default TTL that stores max cap items.
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
		expires: make(map[string]entry),
	}
}

// Set reserves the given key for the configured TTL. It returns false if
// the key is already throttled or the store is full.
func (c *Throttle) Set(key string) bool {
	return c.SetWithTTL(key, c.ttl, 1)
}

// SetWithTTL reserves the given key for the given ttl.
// It returns false if the key is already throttled or the store is full.
func (c *Throttle) SetWithTTL(key string, ttl time.Duration, limit int) bool {
	if limit <= 0 {
		return false
	}
	if ttl <= 0 {
		ttl = c.ttl
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Periodically clean up expired keys.
	now := time.Now()
	if !now.Before(c.cleanAt) {
		for key, e := range c.expires {
			if !now.Before(e.expiry) {
				delete(c.expires, key)
			}
		}

		c.cleanAt = now.Add(min(c.ttl, time.Minute))
	}

	// Key exists and is still within the TTL.
	if e, ok := c.expires[key]; ok {
		if now.Before(e.expiry) {
			if e.count >= limit {
				return false
			}

			e.count++
			c.expires[key] = e

			return true
		}
	} else if len(c.expires) >= c.maxLen {
		return false
	}

	c.expires[key] = entry{expiry: now.Add(ttl), count: 1}

	// New unthrottled key.
	return true
}
