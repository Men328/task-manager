package repository

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"taskmanager/service/attachment/internal/model"
)

func s3TestEnv(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func TestS3BlobStoreRoundTripAgainstLiveEndpoint(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("S3_TEST_ENDPOINT"))
	if endpoint == "" {
		t.Skip("đặt S3_TEST_ENDPOINT để chạy integration test với server S3-compatible")
	}

	store, err := NewS3BlobStore(
		endpoint,
		s3TestEnv("S3_TEST_ACCESS_KEY", "rustfsadmin"),
		s3TestEnv("S3_TEST_SECRET_KEY", "rustfsadmin"),
		s3TestEnv("S3_TEST_BUCKET", "task-manager"),
		s3TestEnv("S3_TEST_REGION", "us-east-1"),
		false,
		10*time.Second,
	)
	if err != nil {
		t.Fatalf("tạo S3 store: %v", err)
	}

	ctx := context.Background()
	if err := store.EnsureBucket(ctx); err != nil {
		t.Fatalf("ensure bucket: %v", err)
	}

	key := "attachments/test/round-trip.txt"
	payload := []byte("hello rustfs")
	if err := store.Put(ctx, key, "text/plain", payload); err != nil {
		t.Fatalf("put object: %v", err)
	}

	data, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("object content = %q, want %q", string(data), string(payload))
	}

	if err := store.Remove(ctx, key); err != nil {
		t.Fatalf("remove object: %v", err)
	}
	if _, err := store.Get(ctx, key); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected not found after remove, got %v", err)
	}
}
