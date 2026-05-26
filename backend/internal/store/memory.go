package store

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/compose"
)

// NewInMemoryStore returns a simple in-memory CheckPointStore.
func NewInMemoryStore() compose.CheckPointStore {
	return &inMemoryStore{mem: map[string][]byte{}}
}

type inMemoryStore struct {
	mu  sync.RWMutex
	mem map[string][]byte
}

func (s *inMemoryStore) Set(_ context.Context, key string, value []byte) error {
	s.mu.Lock()
	s.mem[key] = value
	s.mu.Unlock()
	return nil
}

func (s *inMemoryStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	s.mu.RLock()
	v, ok := s.mem[key]
	s.mu.RUnlock()
	return v, ok, nil
}
