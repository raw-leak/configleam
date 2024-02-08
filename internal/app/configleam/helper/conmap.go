package helper

import "sync"

// ConcurrentMap is a struct that allows adding keys to a map from multiple goroutines executed in parallel.
type ConcurrentMap[T interface{}] struct {
	m    map[string]T
	lock sync.RWMutex
}

// NewConcurrentMap initializes a new ConcurrentMap.
func NewConcurrentMap[T any]() *ConcurrentMap[T] {
	return &ConcurrentMap[T]{
		m: make(map[string]T),
	}
}

// Set sets a key-value pair in the map.
func (cm *ConcurrentMap[T]) Set(key string, value T) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	cm.m[key] = value
}

// Get retrieves a value for a given key from the map.
func (cm *ConcurrentMap[T]) Get(key string) (T, bool) {
	cm.lock.RLock()
	defer cm.lock.RUnlock()
	value, ok := cm.m[key]
	return value, ok
}

// Get full map.
func (cm *ConcurrentMap[T]) GetMap() map[string]T {
	cm.lock.RLock()
	defer cm.lock.RUnlock()
	return cm.m
}
