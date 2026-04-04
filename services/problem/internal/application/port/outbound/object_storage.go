package outbound

import (
	"context"
	"time"
)

// ObjectInfo represents metadata of an object in storage.
type ObjectInfo struct {
	Key          string
	LastModified time.Time
}

type ObjectStorage interface {
	// UploadFromFile streams a local file to MinIO (RAM-safe)
	UploadFromFile(ctx context.Context, objectKey string, filePath string) error
	// DeleteObject removes an object from MinIO
	DeleteObject(ctx context.Context, objectKey string) error
	// GetPresignedURL generates a temporary download URL
	GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	// ListObjectsByPrefix lists all object keys with given prefix
	ListObjectsByPrefix(ctx context.Context, prefix string) ([]string, error)
	// ListObjectsWithInfo lists all objects with metadata (key + lastModified) for GC
	ListObjectsWithInfo(ctx context.Context, prefix string) ([]ObjectInfo, error)
	// EnsureBucket creates bucket if not exists
	EnsureBucket(ctx context.Context) error
}
