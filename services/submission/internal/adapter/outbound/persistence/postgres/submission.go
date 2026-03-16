package postgres

import (
	"context"
	"time"

	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain/entity"

	"gorm.io/gorm"
)

type SubmissionDAO struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	ProblemID     int64     `gorm:"not null;index"`
	UserID        string    `gorm:"not null;size:100;index"`
	Username      string    `gorm:"not null;size:255"`
	Language      string    `gorm:"type:varchar(20);not null;index"`
	SourceCode    string    `gorm:"type:text;not null"`
	Status        string    `gorm:"type:varchar(30);not null;index"`
	ExecutionTime *int      `gorm:"type:int"`
	MemoryUsed    *int      `gorm:"type:int"`
	Score         *float64  `gorm:"type:numeric(5,2)"`
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

func toSubmissionDAO(s *entity.Submission) *SubmissionDAO {
	return &SubmissionDAO{
		ID:            s.ID,
		ProblemID:     s.ProblemID,
		UserID:        s.UserID,
		Username:      s.Username,
		Language:      string(s.Language),
		SourceCode:    s.SourceCode,
		Status:        string(s.Status),
		ExecutionTime: s.ExecutionTime,
		MemoryUsed:    s.MemoryUsed,
		Score:         s.Score,
		CompileOutput: s.CompileOutput,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}
