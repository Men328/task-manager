package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"taskmanager/service/attachment/internal/model"
)

type s3BlobStore struct {
	client  *minio.Client
	bucket  string
	region  string
	timeout time.Duration
}

func NewS3BlobStore(endpoint string, accessKey string, secretKey string, bucket string, region string, useSSL bool, timeout time.Duration) (*s3BlobStore, error) {
	if strings.TrimSpace(bucket) == "" {
		bucket = "task-manager"
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client, err := minio.New(strings.TrimSpace(endpoint), &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrStorageFailed, err)
	}
	return &s3BlobStore{client: client, bucket: bucket, region: region, timeout: timeout}, nil
}

func (s *s3BlobStore) EnsureBucket(ctx context.Context) error {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	exists, err := s.client.BucketExists(callCtx, s.bucket)
	if err != nil {
		return fmt.Errorf("%w: %v", model.ErrStorageFailed, err)
	}
	if exists {
		return nil
	}

	if err := s.client.MakeBucket(callCtx, s.bucket, minio.MakeBucketOptions{Region: s.region}); err != nil {
		if retry, retryErr := s.client.BucketExists(callCtx, s.bucket); retryErr == nil && retry {
			return nil
		}
		return fmt.Errorf("%w: %v", model.ErrStorageFailed, err)
	}
	return nil
}

func (s *s3BlobStore) Put(ctx context.Context, key string, contentType string, data []byte) error {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.client.PutObject(callCtx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("%w: %v", model.ErrStorageFailed, err)
	}
	return nil
}

func (s *s3BlobStore) Get(ctx context.Context, key string) ([]byte, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	object, err := s.client.GetObject(callCtx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrStorageFailed, err)
	}
	defer func() { _ = object.Close() }()

	data, err := io.ReadAll(object)
	if err != nil {
		if isObjectMissing(err) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("%w: %v", model.ErrStorageFailed, err)
	}
	return data, nil
}

func (s *s3BlobStore) Remove(ctx context.Context, key string) error {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := s.client.RemoveObject(callCtx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("%w: %v", model.ErrStorageFailed, err)
	}
	return nil
}

func isObjectMissing(err error) bool {
	var response minio.ErrorResponse
	if errors.As(err, &response) {
		return response.Code == "NoSuchKey" || response.StatusCode == 404
	}
	return false
}
