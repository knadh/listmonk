// Package tmptokens provides a simple in memory store for one time temp tokens with TTL.
// This can be used for creating throwaway tokens for flows like password reset, 2FA verification, etc.
// Tokens are automatically deleted when retrieved or when they expire.
package tmptokens

import (
	"errors"
	"sync"
	"time"
)

var (
	Err = errors.New("token was not found or already expired")
)

// Token represents a temporary token with TTL and arbitrary data.
type Token struct {
	TTL       time.Duration
	CreatedAt time.Time
	Data      any
}

var (
	tokens = make(map[string]Token)
	mu     sync.RWMutex
)

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
func Check(id string) (any, error) {
	mu.RLock()
	defer mu.RUnlock()

	token, exists := tokens[id]
	if !exists {
		return nil, Err
	}

	// Check if token has expired.
	if time.Since(token.CreatedAt) > token.TTL {
		return nil, Err
	}

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
// to prevent memory leaks from tokens that were never retrieved.
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
