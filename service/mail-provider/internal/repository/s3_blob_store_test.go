package repository

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

func s3TestEnv(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func TestS3BlobStorePutAgainstLiveEndpoint(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("S3_TEST_ENDPOINT"))
	if endpoint == "" {
		t.Skip("đặt S3_TEST_ENDPOINT để chạy integration test với server S3-compatible")
	}

	store, err := NewS3BlobStore(
		endpoint,
		s3TestEnv("S3_TEST_ACCESS_KEY", "rustfsadmin"),
		s3TestEnv("S3_TEST_SECRET_KEY", "rustfsadmin"),
		s3TestEnv("S3_TEST_BUCKET", "task-manager-test"),
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

	key := "mail/test/profile/20261003/m1.json"
	payload := []byte(`{"id":"m1"}`)
	stored, err := store.Put(ctx, key, "application/json", payload)
	if err != nil {
		t.Fatalf("put object: %v", err)
	}
	if stored != key {
		t.Fatalf("object key sai: %q", stored)
	}

	info, err := store.client.StatObject(ctx, store.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		t.Fatalf("stat object: %v", err)
	}
	if info.Size != int64(len(payload)) {
		t.Fatalf("kích thước object sai: %d", info.Size)
	}

	if err := store.client.RemoveObject(ctx, store.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		t.Fatalf("xoá object: %v", err)
	}
}
