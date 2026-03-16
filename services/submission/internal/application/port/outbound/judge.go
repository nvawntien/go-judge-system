package outbound

import (
	"context"
	"go-judge-system/services/submission/internal/domain/entity"
)

type JudgePublisher interface {
	Publish(ctx context.Context, submission *entity.Submission) error
}
