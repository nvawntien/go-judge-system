package testcase

import (
	"context"
	"time"

	"go-judge-system/services/problem/internal/application/port/outbound"

	"go.uber.org/zap"
)

const (
	// gcGracePeriod is the minimum age an orphan object must have before deletion.
	// This prevents deleting objects that were just uploaded but not yet committed to DB.
	gcGracePeriod = 24 * time.Hour

	// gcObjectPrefix is the MinIO prefix to scan for testcase ZIP objects.
	gcObjectPrefix = "problems/"
)

type gcOrphanZipsUseCase struct {
	tcRepo  outbound.TestCaseRepository
	storage outbound.ObjectStorage
	logger  *zap.Logger
}

func NewGCOrphanZipsUseCase(tcRepo outbound.TestCaseRepository, storage outbound.ObjectStorage, logger *zap.Logger) *gcOrphanZipsUseCase {
	return &gcOrphanZipsUseCase{tcRepo: tcRepo, storage: storage, logger: logger}
}

// Execute scans MinIO for orphan ZIP objects that are no longer referenced by any
// test_cases DB record. Objects older than gcGracePeriod (24h) are deleted.
//
// This is safe to run concurrently — worst case, an orphan survives one extra cycle.
func (uc *gcOrphanZipsUseCase) Execute(ctx context.Context) error {
	// Ensure the bucket exists before sweeping
	if err := uc.storage.EnsureBucket(ctx); err != nil {
		uc.logger.Error("gc: failed to ensure bucket exists", zap.Error(err))
		return err
	}

	// Get all active zip_object_keys from DB
	activeKeys, err := uc.tcRepo.ListAllZipObjectKeys(ctx)
	if err != nil {
		uc.logger.Error("gc: failed to list active keys from DB", zap.Error(err))
		return err
	}

	activeSet := make(map[string]struct{}, len(activeKeys))
	for _, key := range activeKeys {
		activeSet[key] = struct{}{}
	}

	// List all objects in MinIO with prefix "problems/"
	objects, err := uc.storage.ListObjectsWithInfo(ctx, gcObjectPrefix)
	if err != nil {
		uc.logger.Error("gc: failed to list objects from MinIO", zap.Error(err))
		return err
	}

	// 3. Find orphans = objects not in activeSet && older than grace period
	now := time.Now()
	deletedCount := 0
	skippedCount := 0

	for _, obj := range objects {
		if _, isActive := activeSet[obj.Key]; isActive {
			continue // still referenced by DB
		}

		age := now.Sub(obj.LastModified)
		if age < gcGracePeriod {
			skippedCount++
			uc.logger.Debug("gc: orphan too young, skipping",
				zap.String("key", obj.Key),
				zap.Duration("age", age),
			)
			continue
		}

		// Delete orphan
		if err := uc.storage.DeleteObject(ctx, obj.Key); err != nil {
			uc.logger.Error("gc: failed to delete orphan object",
				zap.String("key", obj.Key),
				zap.Error(err),
			)
			continue // don't abort — try remaining objects
		}

		deletedCount++
		uc.logger.Info("gc: deleted orphan object",
			zap.String("key", obj.Key),
			zap.Duration("age", age),
		)
	}

	uc.logger.Info("gc: sweep completed",
		zap.Int("active_keys", len(activeKeys)),
		zap.Int("total_objects", len(objects)),
		zap.Int("deleted", deletedCount),
		zap.Int("skipped_young", skippedCount),
	)

	return nil
}
