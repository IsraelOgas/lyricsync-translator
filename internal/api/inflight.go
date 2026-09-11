package api

import "sync"

// inflightSet tracks keys that currently have work in flight. Thread-safe.
type inflightSet struct {
	mu    sync.Mutex
	items map[string]bool
}

func newInflightSet() *inflightSet {
	return &inflightSet{items: make(map[string]bool)}
}

// TryAdd marks key as in-flight. Returns false if it was already in-flight.
func (s *inflightSet) TryAdd(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.items[key] {
		return false
	}
	s.items[key] = true
	return true
}

// Remove clears the in-flight mark for key. Safe to call multiple times.
func (s *inflightSet) Remove(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
}
