// Package tmptokens provides a simple in memory store for one time temp tokens with TTL.
// This can be used for creating throwaway tokens for flows like password reset, 2FA verification, etc.
// Tokens are automatically deleted when retrieved or when they expire.
package tmptokens

import (
	"errors"
	"sync"
	"time"
)

const (
	// maxTries is the maximum number of verification attempts allowed for a token.
	// After this many failed checks, the token is automatically deleted.
	maxTries = 15
)

// Token represents a temporary token with TTL and arbitrary data.
type Token struct {
	TTL       time.Duration
	CreatedAt time.Time
	Count     int
	Data      any
}

var (
	Err = errors.New("token was not found or has expired")

	tokens = make(map[string]Token)
	mu     sync.RWMutex
)

func init() {
	// Start periodic cleanup of expired temporary tokens (2FA, password reset).
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			Clean()
		}
	}()
}

// Set stores a token with the given ID, TTL, and data.
// If a token with the same ID already exists, it will be overwritten silently.
func Set(id string, ttl time.Duration, data any) {
	mu.Lock()
	defer mu.Unlock()

	tokens[id] = Token{
		TTL:       ttl,
		Data:      data,
		CreatedAt: time.Now(),
	}
}

// Check retrieves a token by ID without deleting it.
// An error is returned if the token doesn't exist or has expired.
// Unlike Get(), this method does not consume/delete the token.
// It also increments the check counter and deletes the token if maxTries is exceeded,
// acting as a rate limiter.
func Check(id string) (any, error) {
	mu.Lock()
	defer mu.Unlock()

	token, exists := tokens[id]
	if !exists {
		return nil, Err
	}

	// Check if token has expired.
	if time.Since(token.CreatedAt) > token.TTL {
		delete(tokens, id)
		return nil, Err
	}

	// Increment the rate limit counter.
	token.Count++

	// Check if max attempts exceeded.
	if token.Count > maxTries {
		delete(tokens, id)
		return nil, Err
	}

	// Update the token with the new count.
	tokens[id] = token

	return token.Data, nil
}

// Get retrieves a token by ID and automatically deletes it (after one time use).
// An error is returned if the token doesn't exist or has expired.
func Get(id string) (any, error) {
	mu.Lock()
	defer mu.Unlock()

	token, exists := tokens[id]
	if !exists {
		return nil, Err
	}

	// Check if token has expired.
	if time.Since(token.CreatedAt) > token.TTL {
		delete(tokens, id)
		return nil, Err
	}

	// Delete the token.
	delete(tokens, id)

	return token.Data, nil
}

// Delete deletes a token by ID.
func Delete(id string) {
	mu.Lock()
	defer mu.Unlock()

	delete(tokens, id)
}

// Clean deletes all expired tokens. This can be called periodically
// to purge unused and expired tokens.
func Clean() {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	for id, token := range tokens {
		if now.Sub(token.CreatedAt) > token.TTL {
			delete(tokens, id)
		}
	}
}
