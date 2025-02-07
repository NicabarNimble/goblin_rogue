package token

import (
	"context"
	"sync"
)

// MemoryStorage provides an in-memory implementation of the Storage interface
// This implementation is primarily intended for testing and as a reference
// implementation. It is not recommended for production use as tokens are
// lost when the program exits.
type MemoryStorage struct {
	mu     sync.RWMutex
	tokens map[string]Token
}

// NewMemoryStorage creates a new instance of MemoryStorage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		tokens: make(map[string]Token),
	}
}

// Store implements Storage.Store
func (m *MemoryStorage) Store(_ context.Context, key string, token Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Only check for empty value here, allow expired tokens to be stored
	if token.Value == "" {
		return ErrTokenInvalid
	}

	m.tokens[key] = token
	return nil
}

// Retrieve implements Storage.Retrieve
func (m *MemoryStorage) Retrieve(_ context.Context, key string) (Token, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	token, exists := m.tokens[key]
	if !exists {
		return Token{}, ErrTokenNotFound
	}

	if IsExpired(token) {
		return Token{}, ErrTokenExpired
	}

	return token, nil
}

// Delete implements Storage.Delete
func (m *MemoryStorage) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tokens, key)
	return nil
}

// List implements Storage.List
func (m *MemoryStorage) List(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.tokens))
	for k := range m.tokens {
		keys = append(keys, k)
	}
	return keys, nil
}

// Close implements Storage.Close
func (m *MemoryStorage) Close(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear all tokens
	m.tokens = make(map[string]Token)
	return nil
}
