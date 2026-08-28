package problem

import (
	"context"

	"go-judge-system/services/problem/internal/application/port/outbound"
)

func invalidateSubmissionProblemCache(
	ctx context.Context,
	cache outbound.SubmissionProblemCache,
	problemID int64,
) {
	if cache != nil {
		// Cache invalidation is best effort: PostgreSQL has already committed and
		// the short cache TTL bounds any stale entry if Redis is unavailable.
		_ = cache.Delete(ctx, problemID)
	}
}
