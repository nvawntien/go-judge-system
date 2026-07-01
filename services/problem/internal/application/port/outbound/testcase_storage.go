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

type TestCaseStorage interface {
	// UploadFromFile streams a local file to MinIO (RAM-safe)
	UploadTestCase(ctx context.Context, objectKey string, filePath string) error
	// DeleteTestCase removes a test case from MinIO
	DeleteTestCase(ctx context.Context, objectKey string) error
	// GetPresignedURL generates a temporary download URL
	GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	// ListTestCasesByPrefix lists all test case keys with given prefix
	ListTestCasesByPrefix(ctx context.Context, prefix string) ([]string, error)
	// ListTestCaseWithInfo lists all test cases with metadata (key + lastModified) for GC
	ListTestCaseWithInfo(ctx context.Context, prefix string) ([]ObjectInfo, error)
	// EnsureBucket creates bucket if not exists
	EnsureBucket(ctx context.Context) error
}
