package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain/entity"
)

type failingUpdateProblemRepository struct {
	*accessMatrixProblemRepository
}

func (r *failingUpdateProblemRepository) Update(context.Context, *entity.Problem) error {
	return errors.New("database unavailable")
}

type recordingSubmissionProblemCache struct {
	deleted   []int64
	deleteErr error
}

func (c *recordingSubmissionProblemCache) Get(context.Context, int64) (outbound.SubmissionProblemMetadata, bool, error) {
	return outbound.SubmissionProblemMetadata{}, false, nil
}

func (c *recordingSubmissionProblemCache) Set(context.Context, outbound.SubmissionProblemMetadata, time.Duration) error {
	return nil
}

func (c *recordingSubmissionProblemCache) Delete(_ context.Context, problemID int64) error {
	c.deleted = append(c.deleted, problemID)
	return c.deleteErr
}

func TestSubmissionProblemCacheInvalidatedAfterSuccessfulMutations(t *testing.T) {
	claims := matrixClaims(rbac.RoleModerator, "moderator")
	description := "Updated description"

	tests := []struct {
		name string
		run  func(*accessMatrixProblemRepository, *recordingSubmissionProblemCache) error
	}{
		{
			name: "update",
			run: func(repo *accessMatrixProblemRepository, cache *recordingSubmissionProblemCache) error {
				_, err := NewCachedUpdateProblemUseCase(repo, nil, cache).Execute(
					context.Background(), claims, dto.ProblemIDRequest{ID: 42}, dto.UpdateProblemRequest{Description: &description},
				)
				return err
			},
		},
		{
			name: "publish",
			run: func(repo *accessMatrixProblemRepository, cache *recordingSubmissionProblemCache) error {
				_, err := NewCachedPublishProblemUseCase(repo, cache).Execute(context.Background(), claims, dto.ProblemIDRequest{ID: 42})
				return err
			},
		},
		{
			name: "hide",
			run: func(repo *accessMatrixProblemRepository, cache *recordingSubmissionProblemCache) error {
				_, err := NewCachedHiddenProblemUseCase(repo, cache).Execute(context.Background(), claims, dto.ProblemIDRequest{ID: 42})
				return err
			},
		},
		{
			name: "delete",
			run: func(repo *accessMatrixProblemRepository, cache *recordingSubmissionProblemCache) error {
				return NewCachedDeleteProblemUseCase(repo, cache).Execute(context.Background(), claims, dto.ProblemIDRequest{ID: 42})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &accessMatrixProblemRepository{problem: matrixProblem("owner", true)}
			cache := &recordingSubmissionProblemCache{}
			if err := tt.run(repo, cache); err != nil {
				t.Fatalf("mutation error = %v", err)
			}
			if len(cache.deleted) != 1 || cache.deleted[0] != 42 {
				t.Fatalf("cache invalidations = %v, want [42]", cache.deleted)
			}
		})
	}
}

func TestSubmissionProblemCacheIsNotInvalidatedWhenMutationFails(t *testing.T) {
	description := "Updated description"
	cache := &recordingSubmissionProblemCache{}
	repo := &failingUpdateProblemRepository{
		accessMatrixProblemRepository: &accessMatrixProblemRepository{problem: matrixProblem("owner", true)},
	}

	_, err := NewCachedUpdateProblemUseCase(repo, nil, cache).Execute(
		context.Background(),
		matrixClaims(rbac.RoleModerator, "moderator"),
		dto.ProblemIDRequest{ID: 42},
		dto.UpdateProblemRequest{Description: &description},
	)
	if err == nil {
		t.Fatal("mutation error = nil, want error")
	}
	if len(cache.deleted) != 0 {
		t.Fatalf("cache invalidations = %v, want none after failed mutation", cache.deleted)
	}
}

func TestSubmissionProblemCacheInvalidationFailureDoesNotFailCommittedMutation(t *testing.T) {
	cache := &recordingSubmissionProblemCache{deleteErr: errors.New("redis unavailable")}
	repo := &accessMatrixProblemRepository{problem: matrixProblem("owner", false)}

	_, err := NewCachedHiddenProblemUseCase(repo, cache).Execute(
		context.Background(),
		matrixClaims(rbac.RoleModerator, "moderator"),
		dto.ProblemIDRequest{ID: 42},
	)
	if err != nil {
		t.Fatalf("mutation error = %v, want committed mutation success", err)
	}
	if len(cache.deleted) != 1 || cache.deleted[0] != 42 {
		t.Fatalf("cache invalidations = %v, want [42]", cache.deleted)
	}
	if repo.updated == nil || !repo.updated.IsHidden {
		t.Fatalf("updated problem = %+v, want committed hidden state", repo.updated)
	}
}
