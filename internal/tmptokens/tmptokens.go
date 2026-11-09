// Package tmptokens provides a simple in memory store for one time temp tokens with TTL.
// This can be used for creating throwaway tokens for flows like password reset, 2FA verification, etc.
// Tokens are automatically deleted when retrieved or when they expire.
package tmptokens

import (
	"crypto/subtle"
	"errors"
	"sync"
	"time"
)

const (
	// maxTries is the maximum number of verification attempts allowed for a token.
	// Further checks are rejected until the token expires.
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

// SetNX stores a token only if the ID does not exist or has expired.
// Returns data eitherway.
func SetNX(id string, ttl time.Duration, data any) any {
	mu.Lock()
	defer mu.Unlock()

	if token, ok := tokens[id]; ok && time.Since(token.CreatedAt) <= token.TTL {
		return token.Data
	}

	tokens[id] = Token{
		TTL:       ttl,
		CreatedAt: time.Now(),
		Data:      data,
	}
	return data
}

// Match checks a token without increasing attempt counts. If consume is true,
// matching token is deleted atomically; a mismatch leaves it unchanged.
func Match(id, value string, consume bool) bool {
	mu.Lock()
	defer mu.Unlock()

	token, ok := tokens[id]
	if !ok {
		return false
	}

	// Expired.
	if time.Since(token.CreatedAt) > token.TTL {
		delete(tokens, id)
		return false
	}

	stored, ok := token.Data.(string)
	if !ok || subtle.ConstantTimeCompare([]byte(stored), []byte(value)) != 1 {
		return false
	}

	if consume {
		delete(tokens, id)
	}

	return true
}

// Check retrieves a token by ID without deleting it.
// An error is returned if the token doesn't exist or has expired.
// Unlike Get(), this method does not consume/delete the token.
// It allows at most maxTries checks. Exhausted tokens remain until expiry so
// SetNX cannot recreate them and reset the attempt count.
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

	// Retain exhausted tokens until expiry to prevent reset attacks.
	if token.Count >= maxTries {
		return nil, Err
	}
	token.Count++

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
