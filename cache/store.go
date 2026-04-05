package cache

import (
	"sync"
	"time"
)

// Store defines the interface for cache backends.
type Store interface {
	Get(key string) (any, bool)
	Set(key string, value any, ttl time.Duration)
	Delete(key string)
	Clear()
}

// MemoryStore is an in-memory cache implementation.
type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

type cacheItem struct {
	value     any
	expiresAt time.Time
	hasTTL    bool
}

// NewMemoryStore creates a new in-memory cache store.
func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{items: make(map[string]*cacheItem)}
	go s.cleanupLoop()
	return s
}

func (s *MemoryStore) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[key]
	if !ok {
		return nil, false
	}
	if item.hasTTL && time.Now().After(item.expiresAt) {
		return nil, false
	}
	return item.value, true
}

func (s *MemoryStore) Set(key string, value any, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := &cacheItem{value: value}
	if ttl > 0 {
		item.hasTTL = true
		item.expiresAt = time.Now().Add(ttl)
	}
	s.items[key] = item
}

func (s *MemoryStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
}

func (s *MemoryStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[string]*cacheItem)
}

func (s *MemoryStore) cleanupLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, item := range s.items {
			if item.hasTTL && now.After(item.expiresAt) {
				delete(s.items, key)
			}
		}
		s.mu.Unlock()
	}
}
