package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SubmissionDAO struct {
	ID               int64     `gorm:"primaryKey;autoIncrement"`
	ProblemID        int64     `gorm:"not null;index"`
	ProblemName      string    `gorm:"not null;size:500"`
	UserID           string    `gorm:"not null;size:100;index"`
	Username         string    `gorm:"not null;size:255"`
	Language         string    `gorm:"type:varchar(20);not null;index"`
	SourceCode       string    `gorm:"type:text;not null"`
	CurrentAttemptID string    `gorm:"column:current_attempt_id;type:varchar(64);index"`
	Status           string    `gorm:"type:varchar(30);not null;index"`
	ExecutionTime    *int      `gorm:"type:int"`
	MemoryUsed       *int      `gorm:"type:int"`
	CompileOutput    *string   `gorm:"type:text"`
	ErrorMessage     *string   `gorm:"type:text"`
	CreatedAt        time.Time `gorm:"autoCreateTime;index"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime"`
}

func (SubmissionDAO) TableName() string { return "submissions" }

type submissionRepository struct {
	db *gorm.DB
}

func NewSubmissionRepository(db *gorm.DB) outbound.SubmissionRepository {
	db.AutoMigrate(&SubmissionDAO{})
	return &submissionRepository{db: db}
}

func (r *submissionRepository) Create(ctx context.Context, submission *entity.Submission) error {
	dao := toSubmissionDAO(submission)
	db := getDB(ctx, r.db)
	if err := db.Create(dao).Error; err != nil {
		return fmt.Errorf("create submission: %w", err)
	}

	submission.ID = dao.ID
	return nil
}

func (r *submissionRepository) GetByID(ctx context.Context, id int64) (*entity.Submission, error) {
	return r.getByID(ctx, id, false)
}

func (r *submissionRepository) GetByIDForUpdate(ctx context.Context, id int64) (*entity.Submission, error) {
	return r.getByID(ctx, id, true)
}

func (r *submissionRepository) getByID(ctx context.Context, id int64, forUpdate bool) (*entity.Submission, error) {
	var dao SubmissionDAO
	db := getDB(ctx, r.db)
	if forUpdate {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := db.First(&dao, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSubmissionNotFound
		}
		return nil, fmt.Errorf("get submission %d: %w", id, err)
	}

	return toSubmissionEntity(&dao), nil
}

func (r *submissionRepository) Update(ctx context.Context, submission *entity.Submission) error {
	updates := map[string]interface{}{
		"status":             string(submission.Status),
		"current_attempt_id": submission.CurrentAttemptID,
		"execution_time":     submission.ExecutionTime,
		"memory_used":        submission.MemoryUsed,
		"compile_output":     submission.CompileOutput,
		"error_message":      submission.ErrorMessage,
		"updated_at":         submission.UpdatedAt,
	}

	db := getDB(ctx, r.db)
	result := db.Model(&SubmissionDAO{}).Where("id = ?", submission.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update submission %d: %w", submission.ID, result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrSubmissionNotFound
	}

	return nil
}

func (r *submissionRepository) List(
	ctx context.Context,
	filter outbound.ListSubmissionsFilter,
) (outbound.ListSubmissionsResult, error) {
	var total int64
	countQuery := applyListFilters(
		r.db.WithContext(ctx).Model(&SubmissionDAO{}),
		filter,
	)
	if err := countQuery.Count(&total).Error; err != nil {
		return outbound.ListSubmissionsResult{}, fmt.Errorf("count submissions: %w", err)
	}

	var daos []SubmissionDAO
	itemQuery := applyListFilters(
		r.db.WithContext(ctx).Model(&SubmissionDAO{}),
		filter,
	)
	if err := itemQuery.
		Select("id", "problem_id", "problem_name", "user_id", "username", "language", "status", "execution_time", "memory_used", "created_at").
		Order("created_at DESC, id DESC").
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&daos).Error; err != nil {
		return outbound.ListSubmissionsResult{}, fmt.Errorf("list submissions: %w", err)
	}

	return outbound.ListSubmissionsResult{
		Items: toSubmissionEntities(daos),
		Total: total,
	}, nil
}

func (r *submissionRepository) ResultSummaries(
	ctx context.Context,
	submissionIDs []int64,
) (map[int64]outbound.SubmissionResultSummary, error) {
	if len(submissionIDs) == 0 {
		return map[int64]outbound.SubmissionResultSummary{}, nil
	}

	type resultSummaryRow struct {
		SubmissionID int64
		Passed       int
		Total        int
	}

	var rows []resultSummaryRow
	if err := r.db.WithContext(ctx).
		Model(&SubmissionResultDAO{}).
		Select(
			"submission_id, "+
				"SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS passed, "+
				"COUNT(*) AS total",
			string(entity.ResultAccepted),
		).
		Where("submission_id IN ?", submissionIDs).
		Group("submission_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("summarize submission results: %w", err)
	}

	summaries := make(map[int64]outbound.SubmissionResultSummary, len(rows))
	for _, row := range rows {
		summaries[row.SubmissionID] = outbound.SubmissionResultSummary{
			SubmissionID: row.SubmissionID,
			Passed:       row.Passed,
			Total:        row.Total,
		}
	}
	return summaries, nil
}

func applyListFilters(
	query *gorm.DB,
	filter outbound.ListSubmissionsFilter,
) *gorm.DB {
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.Language != nil {
		query = query.Where("language = ?", *filter.Language)
	}
	if filter.ProblemID != nil {
		query = query.Where("problem_id = ?", *filter.ProblemID)
	}
	return query
}

func toSubmissionDAO(s *entity.Submission) *SubmissionDAO {
	return &SubmissionDAO{
		ID:               s.ID,
		ProblemID:        s.ProblemID,
		ProblemName:      s.ProblemName,
		UserID:           s.UserID,
		Username:         s.Username,
		Language:         string(s.Language),
		SourceCode:       s.SourceCode,
		CurrentAttemptID: s.CurrentAttemptID,
		Status:           string(s.Status),
		ExecutionTime:    s.ExecutionTime,
		MemoryUsed:       s.MemoryUsed,
		CompileOutput:    s.CompileOutput,
		ErrorMessage:     s.ErrorMessage,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func toSubmissionEntity(dao *SubmissionDAO) *entity.Submission {
	return &entity.Submission{
		ID:               dao.ID,
		ProblemID:        dao.ProblemID,
		ProblemName:      dao.ProblemName,
		UserID:           dao.UserID,
		Username:         dao.Username,
		Language:         entity.Language(dao.Language),
		SourceCode:       dao.SourceCode,
		CurrentAttemptID: dao.CurrentAttemptID,
		Status:           entity.Status(dao.Status),
		ExecutionTime:    dao.ExecutionTime,
		MemoryUsed:       dao.MemoryUsed,
		CompileOutput:    dao.CompileOutput,
		ErrorMessage:     dao.ErrorMessage,
		CreatedAt:        dao.CreatedAt,
		UpdatedAt:        dao.UpdatedAt,
	}
}

func toSubmissionEntities(daos []SubmissionDAO) []*entity.Submission {
	results := make([]*entity.Submission, 0, len(daos))
	for i := range daos {
		results = append(results, toSubmissionEntity(&daos[i]))
	}
	return results
}
