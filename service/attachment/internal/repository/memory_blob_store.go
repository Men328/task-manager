package repository

import (
	"context"
	"sync"

	"taskmanager/service/attachment/internal/model"
)

type memoryBlobStore struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func NewMemoryBlobStore() *memoryBlobStore {
	return &memoryBlobStore{objects: make(map[string][]byte)}
}

func (s *memoryBlobStore) Put(_ context.Context, key string, _ string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	copied := make([]byte, len(data))
	copy(copied, data)
	s.objects[key] = copied
	return nil
}

func (s *memoryBlobStore) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.objects[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	copied := make([]byte, len(data))
	copy(copied, data)
	return copied, nil
}

func (s *memoryBlobStore) Remove(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.objects, key)
	return nil
}
