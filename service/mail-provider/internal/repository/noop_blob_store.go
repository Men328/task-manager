package repository

import "context"

type noopBlobStore struct{}

func NewNoopBlobStore() *noopBlobStore {
	return &noopBlobStore{}
}

func (s *noopBlobStore) Put(_ context.Context, _ string, _ string, _ []byte) (string, error) {
	return "", nil
}
