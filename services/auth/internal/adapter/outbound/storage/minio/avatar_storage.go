package minio

import (
    "context"
    "fmt"
    "io"
    "net/url"
    "strings"
    "time"

    "go-judge-system/pkg/config"
    "go-judge-system/services/auth/internal/application/port/outbound"

    "github.com/google/uuid"
    "github.com/minio/minio-go/v7"
)

type avatarStorage struct {
    client    *minio.Client
    bucket    string
    publicURL string
}

func NewAvatarStorage(client *minio.Client, cfg config.MinIOConfig) (outbound.AvatarStorage, error) {
	storage := &avatarStorage{
		client:    client,
		bucket:    cfg.Bucket,
		publicURL: strings.TrimRight(cfg.PublicURL, "/"),
	}

	ctxInit, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := storage.ensureBucket(ctxInit); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *avatarStorage) UploadAvatar(ctx context.Context, userID string, contentType string, reader io.Reader, size int64) (string, string, error) {
    objectKey := fmt.Sprintf("users/%s/%d-%s%s", userID, time.Now().UnixMilli(), uuid.NewString(), extensionForContentType(contentType))

    if _, err := s.client.PutObject(ctx, s.bucket, objectKey, reader, size, minio.PutObjectOptions{
        ContentType: contentType,
    }); err != nil {
        return "", "", fmt.Errorf("failed to upload avatar: %w", err)
    }

    avatarURL, err := url.JoinPath(s.publicURL, s.bucket, objectKey)
    if err != nil {
        avatarURL = fmt.Sprintf("%s/%s/%s", s.publicURL, s.bucket, objectKey)
    }

    return avatarURL, objectKey, nil
}

func (s *avatarStorage) DeleteAvatar(ctx context.Context, objectKey string) error {
    if objectKey == "" {
        return nil
    }

    if err := s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
        return fmt.Errorf("failed to delete avatar: %w", err)
    }

    return nil
}

func (s *avatarStorage) ensureBucket(ctx context.Context) error {
    exists, err := s.client.BucketExists(ctx, s.bucket)
    if err != nil {
        return fmt.Errorf("failed to check avatar bucket existence: %w", err)
    }
    if exists {
        return nil
    }

    if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
        return fmt.Errorf("failed to create avatar bucket %s: %w", s.bucket, err)
    }

    return nil
}

func extensionForContentType(contentType string) string {
    switch contentType {
    case "image/jpeg":
        return ".jpg"
    case "image/png":
        return ".png"
    case "image/webp":
        return ".webp"
    default:
        return ".bin"
    }
}