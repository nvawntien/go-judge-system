package minio

import (
	"context"
	"fmt"
	"os"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/services/problem/internal/application/port/outbound"

	"github.com/minio/minio-go/v7"
)

type minioStorage struct {
	client *minio.Client
	bucket string
}

func NewMinioStorage(client *minio.Client, cfg config.MinIOConfig) outbound.ObjectStorage {
	return &minioStorage{client: client, bucket: cfg.Bucket}
}

func (m *minioStorage) UploadFromFile(ctx context.Context, objectKey string, filePath string) error {
	if err := m.EnsureBucket(ctx); err != nil {
		return fmt.Errorf("failed to ensure bucket %s: %w", m.bucket, err)
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

	opts := minio.PutObjectOptions{
		ContentType: "application/zip",
	}

	if _, err := m.client.PutObject(ctx, m.bucket, objectKey, file, fileInfo.Size(), opts); err != nil {
		return fmt.Errorf("failed to upload object %s to bucket %s: %w", objectKey, m.bucket, err)
	}

	return nil
}

func (m *minioStorage) DeleteObject(ctx context.Context, objectKey string) error {
	if err := m.client.RemoveObject(ctx, m.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete object %s from bucket %s: %w", objectKey, m.bucket, err)
	}

	return nil
}

func (m *minioStorage) GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	url, err := m.client.PresignedGetObject(ctx, m.bucket, objectKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get presigned URL for object %s from bucket %s: %w", objectKey, m.bucket, err)
	}

	return url.String(), nil
}

func (m *minioStorage) ListObjectsByPrefix(ctx context.Context, prefix string) ([]string, error) {
	var objectKeys []string
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for object := range m.client.ListObjects(ctx, m.bucket, opts) {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects with prefix %s: %w", prefix, object.Err)
		}
		objectKeys = append(objectKeys, object.Key)
	}

	return objectKeys, nil
}

func (m *minioStorage) ListObjectsWithInfo(ctx context.Context, prefix string) ([]outbound.ObjectInfo, error) {
	var objects []outbound.ObjectInfo
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	for object := range m.client.ListObjects(ctx, m.bucket, opts) {
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

func (m *minioStorage) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if exists {
		return nil
	}

	if err := m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("failed to create bucket %s: %w", m.bucket, err)
	}

	return nil
}
