package problem

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"go-judge-system/pkg/rbac"
	inbound "go-judge-system/services/problem/internal/application/port/inbound"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	submissionProblemCacheTTL     = 5 * time.Second
	submissionProblemCacheTimeout = 100 * time.Millisecond
	// The Submission caller currently gives this RPC one second. A bounded
	// initial cache read plus this shared budget leaves roughly 200ms for
	// authorization, mapping, and gRPC response propagation.
	submissionProblemLoadTimeout = 700 * time.Millisecond
)

type getProblemUseCase struct {
	problemRepo outbound.ProblemRepository
}

type cachedGetProblemUseCase struct {
	problemRepo outbound.SubmissionProblemRepository
	cache       outbound.SubmissionProblemCache
	logger      *zap.Logger
	loads       singleflight.Group
}

func NewGetProblemUseCase(problemRepo outbound.ProblemRepository) inbound.GetProblemUseCase {
	return &getProblemUseCase{problemRepo: problemRepo}
}

// NewCachedGetProblemUseCase is the Problem-owned cache-aside read path used
// by Submission Service. Authorization remains request-specific and is never
// stored in cache.
func NewCachedGetProblemUseCase(
	problemRepo outbound.SubmissionProblemRepository,
	cache outbound.SubmissionProblemCache,
	logger *zap.Logger,
) inbound.GetProblemUseCase {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &cachedGetProblemUseCase{problemRepo: problemRepo, cache: cache, logger: logger}
}

func (uc *getProblemUseCase) Execute(
	ctx context.Context,
	req inbound.GetProblemRequest,
) (inbound.ProblemMetadata, error) {
	if req.ProblemID <= 0 {
		return inbound.ProblemMetadata{}, domain.ErrInvalidInput
	}

	actorUserID := strings.TrimSpace(req.ActorUserID)
	if actorUserID == "" || req.ActorRole == "" {
		return inbound.ProblemMetadata{}, domain.ErrActorUnauthenticated
	}
	if req.ActorRole.Level() == 0 {
		return inbound.ProblemMetadata{}, domain.ErrPermissionDenied
	}

	problem, err := uc.problemRepo.GetByID(ctx, req.ProblemID)
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return inbound.ProblemMetadata{}, err
		case errors.Is(err, domain.ErrProblemNotFound):
			return inbound.ProblemMetadata{}, domain.ErrProblemNotFound
		default:
			return inbound.ProblemMetadata{}, domain.ErrInternalServer.Wrap(err)
		}
	}
	if problem == nil {
		return inbound.ProblemMetadata{}, domain.ErrInternalServer
	}
	if problem.IsDeleted() || !canAccessProblem(problem, actorUserID, req.ActorRole) {
		return inbound.ProblemMetadata{}, domain.ErrProblemNotFound
	}

	return inbound.ProblemMetadata{
		ID:          problem.ID,
		Title:       problem.Title,
		Slug:        problem.TitleSlug,
		TimeLimit:   problem.TimeLimit,
		MemoryLimit: problem.MemoryLimit,
	}, nil
}

func (uc *cachedGetProblemUseCase) Execute(
	ctx context.Context,
	req inbound.GetProblemRequest,
) (inbound.ProblemMetadata, error) {
	if err := validateProblemActorRequest(req); err != nil {
		return inbound.ProblemMetadata{}, err
	}

	problem, err := uc.getSubmissionProblem(ctx, req.ProblemID)
	if err != nil {
		return inbound.ProblemMetadata{}, mapSubmissionProblemReadError(err)
	}
	if !canAccessSubmissionProblem(problem, strings.TrimSpace(req.ActorUserID), req.ActorRole) {
		return inbound.ProblemMetadata{}, domain.ErrProblemNotFound
	}

	return inbound.ProblemMetadata{
		ID:          problem.ID,
		Title:       problem.Title,
		Slug:        problem.Slug,
		TimeLimit:   problem.TimeLimit,
		MemoryLimit: problem.MemoryLimit,
	}, nil
}

func (uc *cachedGetProblemUseCase) getSubmissionProblem(
	ctx context.Context,
	problemID int64,
) (outbound.SubmissionProblemMetadata, error) {
	if metadata, found, err := uc.getFromCache(ctx, problemID); err == nil {
		if found {
			uc.logger.Debug("submission problem cache hit", zap.Int64("problem_id", problemID))
			return metadata, nil
		}
		uc.logger.Debug("submission problem cache miss", zap.Int64("problem_id", problemID))
	} else if ctx.Err() != nil {
		return outbound.SubmissionProblemMetadata{}, ctx.Err()
	} else {
		uc.logger.Warn("submission problem cache read failed; falling back to database", zap.Int64("problem_id", problemID), zap.Error(err))
	}

	result := uc.loads.DoChan(submissionProblemSingleflightKey(problemID), func() (interface{}, error) {
		loadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), submissionProblemLoadTimeout)
		defer cancel()

		if metadata, found, err := uc.getFromCache(loadCtx, problemID); err == nil && found {
			return metadata, nil
		} else if err != nil {
			uc.logger.Warn("submission problem cache recheck failed; falling back to database", zap.Int64("problem_id", problemID), zap.Error(err))
		}

		uc.logger.Debug("submission problem database fallback", zap.Int64("problem_id", problemID))
		metadata, err := uc.problemRepo.GetSubmissionProblem(loadCtx, problemID)
		if err != nil {
			return outbound.SubmissionProblemMetadata{}, err
		}
		if err := uc.setInCache(loadCtx, metadata); err != nil {
			uc.logger.Warn("submission problem cache write failed", zap.Int64("problem_id", problemID), zap.Error(err))
		}
		return metadata, nil
	})

	select {
	case <-ctx.Done():
		return outbound.SubmissionProblemMetadata{}, ctx.Err()
	case shared := <-result:
		if shared.Shared {
			uc.logger.Debug("submission problem load coalesced", zap.Int64("problem_id", problemID))
		}
		if shared.Err != nil {
			return outbound.SubmissionProblemMetadata{}, shared.Err
		}
		metadata, ok := shared.Val.(outbound.SubmissionProblemMetadata)
		if !ok {
			return outbound.SubmissionProblemMetadata{}, domain.ErrInternalServer
		}
		return metadata, nil
	}
}

func (uc *cachedGetProblemUseCase) getFromCache(
	ctx context.Context,
	problemID int64,
) (outbound.SubmissionProblemMetadata, bool, error) {
	cacheCtx, cancel := context.WithTimeout(ctx, submissionProblemCacheTimeout)
	defer cancel()
	return uc.cache.Get(cacheCtx, problemID)
}

func (uc *cachedGetProblemUseCase) setInCache(
	ctx context.Context,
	metadata outbound.SubmissionProblemMetadata,
) error {
	cacheCtx, cancel := context.WithTimeout(ctx, submissionProblemCacheTimeout)
	defer cancel()
	return uc.cache.Set(cacheCtx, metadata, submissionProblemCacheTTL)
}

func validateProblemActorRequest(req inbound.GetProblemRequest) error {
	if req.ProblemID <= 0 {
		return domain.ErrInvalidInput
	}
	if strings.TrimSpace(req.ActorUserID) == "" || req.ActorRole == "" {
		return domain.ErrActorUnauthenticated
	}
	if req.ActorRole.Level() == 0 {
		return domain.ErrPermissionDenied
	}
	return nil
}

func mapSubmissionProblemReadError(err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, domain.ErrProblemNotFound):
		return domain.ErrProblemNotFound
	default:
		return domain.ErrInternalServer.Wrap(err)
	}
}

func canAccessSubmissionProblem(problem outbound.SubmissionProblemMetadata, actorUserID string, actorRole rbac.Role) bool {
	return canAccessProblemState(problem.IsHidden, problem.AuthorID, actorUserID, actorRole)
}

func canAccessProblem(problem *entity.Problem, actorUserID string, actorRole rbac.Role) bool {
	return canAccessProblemState(problem.IsHidden, problem.AuthorID, actorUserID, actorRole)
}

func canAccessProblemState(isHidden bool, authorID, actorUserID string, actorRole rbac.Role) bool {
	if !isHidden {
		return true
	}

	switch actorRole {
	case rbac.RoleContributor:
		return authorID == actorUserID
	case rbac.RoleModerator, rbac.RoleAdmin:
		return true
	default:
		return false
	}
}

func submissionProblemSingleflightKey(problemID int64) string {
	return strconv.FormatInt(problemID, 10)
}
