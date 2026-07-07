package outbound

import (
	"context"
	"io"
)

type AvatarStorage interface {
	UploadAvatar(ctx context.Context, userID string, contentType string, reader io.Reader, size int64) (avatarURL string, objectKey string, err error)
	DeleteAvatar(ctx context.Context, objectKey string) error
}
