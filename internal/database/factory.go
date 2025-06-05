package database

import (
	"fmt"
	"sync"
)

// factory implements the AdapterFactory interface
type factory struct {
	adapters map[DatabaseType]func() Adapter
	mu       sync.RWMutex
}

// NewFactory creates a new adapter factory
func NewFactory() AdapterFactory {
	return &factory{
		adapters: make(map[DatabaseType]func() Adapter),
	}
}

// RegisterAdapter registers a new adapter type with the factory
func (f *factory) RegisterAdapter(dbType DatabaseType, creator func() Adapter) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if _, exists := f.adapters[dbType]; exists {
		return fmt.Errorf("adapter for database type %s is already registered", dbType)
	}
	
	f.adapters[dbType] = creator
	return nil
}

// CreateAdapter creates a new adapter instance for the specified database type
func (f *factory) CreateAdapter(dbType DatabaseType) (Adapter, error) {
	f.mu.RLock()
	creator, exists := f.adapters[dbType]
	f.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("no adapter registered for database type: %s", dbType)
	}
	
	return creator(), nil
}

// SupportedTypes returns a list of supported database types
func (f *factory) SupportedTypes() []DatabaseType {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	types := make([]DatabaseType, 0, len(f.adapters))
	for dbType := range f.adapters {
		types = append(types, dbType)
	}
	
	return types
}

// Global factory instance
var globalFactory = NewFactory()

// RegisterAdapter registers an adapter with the global factory
func RegisterAdapter(dbType DatabaseType, creator func() Adapter) error {
	if factory, ok := globalFactory.(*factory); ok {
		return factory.RegisterAdapter(dbType, creator)
	}
	return fmt.Errorf("global factory is not of expected type")
}

// CreateAdapter creates an adapter using the global factory
func CreateAdapter(dbType DatabaseType) (Adapter, error) {
	return globalFactory.CreateAdapter(dbType)
}

// SupportedTypes returns supported types from the global factory
func SupportedTypes() []DatabaseType {
	return globalFactory.SupportedTypes()
}

// GetFactory returns the global factory instance
func GetFactory() AdapterFactory {
	return globalFactory
}

// SetFactory sets a custom factory as the global factory (useful for testing)
func SetFactory(f AdapterFactory) {
	globalFactory = f
}