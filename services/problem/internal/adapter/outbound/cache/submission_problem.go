package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-judge-system/services/problem/internal/application/port/outbound"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const submissionProblemKeyPrefix = "problem:submission:v1:"

type submissionProblemCache struct {
	client *redis.Client
	logger *zap.Logger
}

// submissionProblemCachePayload uses pointers so a syntactically valid but
// incomplete cache entry is rejected instead of changing authorization state.
type submissionProblemCachePayload struct {
	ID          *int64   `json:"id"`
	Title       *string  `json:"title"`
	Slug        *string  `json:"slug"`
	TimeLimit   *float64 `json:"time_limit"`
	MemoryLimit *int     `json:"memory_limit"`
	AuthorID    *string  `json:"author_id"`
	IsHidden    *bool    `json:"is_hidden"`
}

func NewSubmissionProblemCache(client *redis.Client, logger *zap.Logger) outbound.SubmissionProblemCache {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &submissionProblemCache{client: client, logger: logger}
}

func (c *submissionProblemCache) Get(
	ctx context.Context,
	problemID int64,
) (outbound.SubmissionProblemMetadata, bool, error) {
	if c.client == nil {
		return outbound.SubmissionProblemMetadata{}, false, errors.New("submission problem cache client is unavailable")
	}
	encoded, err := c.client.Get(ctx, submissionProblemKey(problemID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return outbound.SubmissionProblemMetadata{}, false, nil
	}
	if err != nil {
		return outbound.SubmissionProblemMetadata{}, false, err
	}

	metadata, err := decodeSubmissionProblemMetadata(problemID, encoded)
	if err != nil {
		return outbound.SubmissionProblemMetadata{}, false, err
	}
	return metadata, true, nil
}

func (c *submissionProblemCache) Set(
	ctx context.Context,
	metadata outbound.SubmissionProblemMetadata,
	ttl time.Duration,
) error {
	if c.client == nil {
		return errors.New("submission problem cache client is unavailable")
	}
	encoded, err := json.Marshal(submissionProblemCachePayload{
		ID:          &metadata.ID,
		Title:       &metadata.Title,
		Slug:        &metadata.Slug,
		TimeLimit:   &metadata.TimeLimit,
		MemoryLimit: &metadata.MemoryLimit,
		AuthorID:    &metadata.AuthorID,
		IsHidden:    &metadata.IsHidden,
	})
	if err != nil {
		return fmt.Errorf("encode submission problem cache entry: %w", err)
	}
	return c.client.Set(ctx, submissionProblemKey(metadata.ID), encoded, ttl).Err()
}

func (c *submissionProblemCache) Delete(ctx context.Context, problemID int64) error {
	if c.client == nil {
		return errors.New("submission problem cache client is unavailable")
	}
	err := c.client.Del(ctx, submissionProblemKey(problemID)).Err()
	if err != nil {
		c.logger.Warn("submission problem cache invalidation failed", zap.Int64("problem_id", problemID), zap.Error(err))
	}
	return err
}

func submissionProblemKey(problemID int64) string {
	return submissionProblemKeyPrefix + strconv.FormatInt(problemID, 10)
}

func decodeSubmissionProblemMetadata(problemID int64, encoded []byte) (outbound.SubmissionProblemMetadata, error) {
	var payload submissionProblemCachePayload
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return outbound.SubmissionProblemMetadata{}, fmt.Errorf("decode submission problem cache entry: %w", err)
	}
	if payload.ID == nil || *payload.ID != problemID ||
		payload.Title == nil || strings.TrimSpace(*payload.Title) == "" ||
		payload.Slug == nil || strings.TrimSpace(*payload.Slug) == "" ||
		payload.TimeLimit == nil || *payload.TimeLimit <= 0 ||
		payload.MemoryLimit == nil || *payload.MemoryLimit <= 0 ||
		payload.AuthorID == nil || strings.TrimSpace(*payload.AuthorID) == "" ||
		payload.IsHidden == nil {
		return outbound.SubmissionProblemMetadata{}, errors.New("invalid submission problem cache entry")
	}

	return outbound.SubmissionProblemMetadata{
		ID:          *payload.ID,
		Title:       *payload.Title,
		Slug:        *payload.Slug,
		TimeLimit:   *payload.TimeLimit,
		MemoryLimit: *payload.MemoryLimit,
		AuthorID:    *payload.AuthorID,
		IsHidden:    *payload.IsHidden,
	}, nil
}
