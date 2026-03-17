package postgres

import (
	"context"
	"time"

	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain/entity"

	"gorm.io/gorm"
)

type SubmissionResultDAO struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	SubmissionID  int64     `gorm:"not null;index"`
	TestCaseID    int64     `gorm:"not null;index"`
	Status        string    `gorm:"type:varchar(30);not null"`
	ActualOutput  *string   `gorm:"type:text"`
	ExecutionTime *int      `gorm:"type:int"`
	MemoryUsed    *int      `gorm:"type:int"`
	Order         int       `gorm:"not null;index"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (SubmissionResultDAO) TableName() string { return "submission_results" }

type submissionResultRepository struct {
	db *gorm.DB
}

func NewSubmissionResultRepository(db *gorm.DB) outbound.SubmissionResultRepository {
	db.AutoMigrate(&SubmissionResultDAO{})
	return &submissionResultRepository{db: db}
}

func (r *submissionResultRepository) GetBySubmissionID(ctx context.Context, submissionID int64) ([]*entity.SubmissionResult, error) {
	var daos []SubmissionResultDAO
	if err := r.db.WithContext(ctx).
		Where("submission_id = ?", submissionID).
		Order("\"order\" ASC").
		Find(&daos).Error; err != nil {
		return nil, err
	}

	results := make([]*entity.SubmissionResult, 0, len(daos))
	for i := range daos {
		results = append(results, toSubmissionResultEntity(&daos[i]))
	}

	return results, nil
}

func toSubmissionResultEntity(dao *SubmissionResultDAO) *entity.SubmissionResult {
	return &entity.SubmissionResult{
		ID:            dao.ID,
		SubmissionID:  dao.SubmissionID,
		TestCaseID:    dao.TestCaseID,
		Status:        entity.ResultStatus(dao.Status),
		ActualOutput:  dao.ActualOutput,
		ExecutionTime: dao.ExecutionTime,
		MemoryUsed:    dao.MemoryUsed,
		Order:         dao.Order,
		CreatedAt:     dao.CreatedAt,
	}
}
