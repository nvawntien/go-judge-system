package testcase

import (
	"context"
	"time"

	"go.uber.org/zap"
)

const gcInterval = 6 * time.Hour

// GCRunner runs the orphan ZIP garbage collection on a periodic interval.
type GCRunner struct {
	gc     *gcOrphanZipsUseCase
	logger *zap.Logger
}

func NewGCRunner(gc *gcOrphanZipsUseCase, logger *zap.Logger) *GCRunner {
	return &GCRunner{gc: gc, logger: logger}
}

// Start runs the GC loop in a background goroutine. It blocks until ctx is cancelled.
// Call this with `go gcRunner.Start(ctx)` from the application entrypoint.
func (r *GCRunner) Start(ctx context.Context) {
	r.logger.Info("gc: background runner started", zap.Duration("interval", gcInterval))

	// Run once immediately on startup
	if err := r.gc.Execute(ctx); err != nil {
		r.logger.Error("gc: initial sweep failed", zap.Error(err))
	}

	ticker := time.NewTicker(gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("gc: background runner stopped")
			return
		case <-ticker.C:
			if err := r.gc.Execute(ctx); err != nil {
				r.logger.Error("gc: sweep failed", zap.Error(err))
			}
		}
	}
}
