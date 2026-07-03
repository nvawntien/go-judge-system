package minio

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/services/problem/internal/application/port/outbound"

	"github.com/minio/minio-go/v7"
)

type testcaseStorage struct {
	client *minio.Client
	bucket string
}

func NewTestCaseStorage(client *minio.Client, cfg config.MinIOConfig) (outbound.TestCaseStorage, error) {
	if client == nil {
		return nil, fmt.Errorf("minio client is nil")
	}

	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, fmt.Errorf("minio bucket is required")
	}

	storage := &testcaseStorage{
		client: client,
		bucket: strings.TrimSpace(cfg.Bucket),
	}

	ctxInit, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := storage.EnsureBucket(ctxInit); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *testcaseStorage) UploadTestCase(ctx context.Context, objectKey string, filePath string) error {
	if strings.TrimSpace(objectKey) == "" {
		return fmt.Errorf("object key is required")
	}

	if strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("file path is required")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	if fileInfo.Size() <= 0 {
		return fmt.Errorf("testcase file must not be empty")
	}

	if _, err := s.client.PutObject(ctx, s.bucket, objectKey, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: "application/zip",
	}); err != nil {
		return fmt.Errorf("failed to upload object %s to bucket %s: %w", objectKey, s.bucket, err)
	}

	return nil
}

func (s *testcaseStorage) DeleteTestCase(ctx context.Context, objectKey string) error {
	if strings.TrimSpace(objectKey) == "" {
		return nil
	}

	if err := s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete object %s from bucket %s: %w", objectKey, s.bucket, err)
	}

	return nil
}

func (s *testcaseStorage) GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	if strings.TrimSpace(objectKey) == "" {
		return "", fmt.Errorf("object key is required")
	}

	url, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL for object %s from bucket %s: %w", objectKey, s.bucket, err)
	}

	return url.String(), nil
}

func (s *testcaseStorage) ListTestCasesByPrefix(ctx context.Context, prefix string) ([]string, error) {
	var objectKeys []string

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for object := range s.client.ListObjects(ctx, s.bucket, opts) {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects with prefix %s: %w", prefix, object.Err)
		}

		objectKeys = append(objectKeys, object.Key)
	}

	return objectKeys, nil
}

func (s *testcaseStorage) ListTestCaseWithInfo(ctx context.Context, prefix string) ([]outbound.ObjectInfo, error) {
	var objects []outbound.ObjectInfo

	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for object := range s.client.ListObjects(ctx, s.bucket, opts) {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects with prefix %s: %w", prefix, object.Err)
		}

		objects = append(objects, outbound.ObjectInfo{
			Key:          object.Key,
			LastModified: object.LastModified,
		})
	}

	return objects, nil
}

func (s *testcaseStorage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if exists {
		return nil
	}

	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", s.bucket, err)
	}

	return nil
}
