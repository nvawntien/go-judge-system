package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"

	"gorm.io/gorm"
)

type SubmissionDAO struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	ProblemID     int64     `gorm:"not null;index"`
	ProblemName   string    `gorm:"not null;size:500"`
	UserID        string    `gorm:"not null;size:100;index"`
	Username      string    `gorm:"not null;size:255"`
	Language      string    `gorm:"type:varchar(20);not null;index"`
	SourceCode    string    `gorm:"type:text;not null"`
	Status        string    `gorm:"type:varchar(30);not null;index"`
	ExecutionTime *int      `gorm:"type:int"`
	MemoryUsed    *int      `gorm:"type:int"`
	CompileOutput *string   `gorm:"type:text"`
	CreatedAt     time.Time `gorm:"autoCreateTime;index"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
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
	var dao SubmissionDAO
	db := getDB(ctx, r.db)
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
		"status":         string(submission.Status),
		"execution_time": submission.ExecutionTime,
		"memory_used":    submission.MemoryUsed,
		"compile_output": submission.CompileOutput,
		"updated_at":     submission.UpdatedAt,
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

func (r *submissionRepository) ListByUser(
	ctx context.Context,
	filter outbound.ListSubmissionsFilter,
) (outbound.ListSubmissionsResult, error) {
	if strings.TrimSpace(filter.UserID) == "" {
		return outbound.ListSubmissionsResult{}, fmt.Errorf("list submissions by user: user ID is required")
	}

	var total int64
	countQuery := applyListByUserFilters(
		r.db.WithContext(ctx).Model(&SubmissionDAO{}),
		filter,
	)
	if err := countQuery.Count(&total).Error; err != nil {
		return outbound.ListSubmissionsResult{}, fmt.Errorf("count submissions by user: %w", err)
	}

	var daos []SubmissionDAO
	itemQuery := applyListByUserFilters(
		r.db.WithContext(ctx).Model(&SubmissionDAO{}),
		filter,
	)
	if err := itemQuery.
		Select("id", "problem_id", "problem_name", "language", "status", "created_at").
		Order("created_at DESC, id DESC").
		Offset(filter.Offset).
		Limit(filter.Limit).
		Find(&daos).Error; err != nil {
		return outbound.ListSubmissionsResult{}, fmt.Errorf("list submissions by user: %w", err)
	}

	return outbound.ListSubmissionsResult{
		Items: toSubmissionEntities(daos),
		Total: total,
	}, nil
}

func (r *submissionRepository) ListByProblem(ctx context.Context, problemID int64, offset, limit int, status, language string) ([]*entity.Submission, error) {
	query := r.db.WithContext(ctx).Where("problem_id = ?", problemID)
	query = applyListFilters(query, status, language)

	var daos []SubmissionDAO
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&daos).Error; err != nil {
		return nil, fmt.Errorf("list submissions by problem: %w", err)
	}

	return toSubmissionEntities(daos), nil
}

func (r *submissionRepository) CountByProblem(ctx context.Context, problemID int64, status, language string) (int64, error) {
	query := r.db.WithContext(ctx).Model(&SubmissionDAO{}).Where("problem_id = ?", problemID)
	query = applyListFilters(query, status, language)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count submissions by problem: %w", err)
	}
	return count, nil
}

func (r *submissionRepository) ListAll(ctx context.Context, offset, limit int, problemID *int64, userID, status, language string) ([]*entity.Submission, error) {
	query := r.db.WithContext(ctx).Model(&SubmissionDAO{})
	query = applyAdminFilters(query, problemID, userID, status, language)

	var daos []SubmissionDAO
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&daos).Error; err != nil {
		return nil, fmt.Errorf("list submissions: %w", err)
	}

	return toSubmissionEntities(daos), nil
}

func (r *submissionRepository) CountAll(ctx context.Context, problemID *int64, userID, status, language string) (int64, error) {
	query := r.db.WithContext(ctx).Model(&SubmissionDAO{})
	query = applyAdminFilters(query, problemID, userID, status, language)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count submissions: %w", err)
	}
	return count, nil
}

func applyListFilters(query *gorm.DB, status, language string) *gorm.DB {
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if language != "" {
		query = query.Where("language = ?", language)
	}
	return query
}

func applyListByUserFilters(
	query *gorm.DB,
	filter outbound.ListSubmissionsFilter,
) *gorm.DB {
	query = query.Where("user_id = ?", filter.UserID)
	query = applyListFilters(query, filter.Status, filter.Language)
	if filter.ProblemID != nil {
		query = query.Where("problem_id = ?", *filter.ProblemID)
	}
	return query
}

func applyAdminFilters(query *gorm.DB, problemID *int64, userID, status, language string) *gorm.DB {
	if problemID != nil {
		query = query.Where("problem_id = ?", *problemID)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	return applyListFilters(query, status, language)
}

func toSubmissionDAO(s *entity.Submission) *SubmissionDAO {
	return &SubmissionDAO{
		ID:            s.ID,
		ProblemID:     s.ProblemID,
		ProblemName:   s.ProblemName,
		UserID:        s.UserID,
		Username:      s.Username,
		Language:      string(s.Language),
		SourceCode:    s.SourceCode,
		Status:        string(s.Status),
		ExecutionTime: s.ExecutionTime,
		MemoryUsed:    s.MemoryUsed,
		CompileOutput: s.CompileOutput,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func toSubmissionEntity(dao *SubmissionDAO) *entity.Submission {
	return &entity.Submission{
		ID:            dao.ID,
		ProblemID:     dao.ProblemID,
		ProblemName:   dao.ProblemName,
		UserID:        dao.UserID,
		Username:      dao.Username,
		Language:      entity.Language(dao.Language),
		SourceCode:    dao.SourceCode,
		Status:        entity.Status(dao.Status),
		ExecutionTime: dao.ExecutionTime,
		MemoryUsed:    dao.MemoryUsed,
		CompileOutput: dao.CompileOutput,
		CreatedAt:     dao.CreatedAt,
		UpdatedAt:     dao.UpdatedAt,
	}
}

func toSubmissionEntities(daos []SubmissionDAO) []*entity.Submission {
	results := make([]*entity.Submission, 0, len(daos))
	for i := range daos {
		results = append(results, toSubmissionEntity(&daos[i]))
	}
	return results
}
