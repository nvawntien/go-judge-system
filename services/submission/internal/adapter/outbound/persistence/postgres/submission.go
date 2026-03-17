package postgres

import (
	"context"
	"errors"
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
	if err := r.db.WithContext(ctx).Create(dao).Error; err != nil {
		return err
	}

	submission.ID = dao.ID
	return nil
}

func (r *submissionRepository) GetByID(ctx context.Context, id int64) (*entity.Submission, error) {
	var dao SubmissionDAO
	if err := r.db.WithContext(ctx).First(&dao, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSubmissionNotFound
		}
		return nil, err
	}

	return toSubmissionEntity(&dao), nil
}

func (r *submissionRepository) ListByUser(ctx context.Context, userID string, offset, limit int, status, language string) ([]*entity.Submission, error) {
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	query = applyListFilters(query, status, language)

	var daos []SubmissionDAO
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&daos).Error; err != nil {
		return nil, err
	}

	return toSubmissionEntities(daos), nil
}

func (r *submissionRepository) CountByUser(ctx context.Context, userID string, status, language string) (int64, error) {
	query := r.db.WithContext(ctx).Model(&SubmissionDAO{}).Where("user_id = ?", userID)
	query = applyListFilters(query, status, language)

	var count int64
	return count, query.Count(&count).Error
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
