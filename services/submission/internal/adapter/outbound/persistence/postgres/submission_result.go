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
	TestIndex     int       `gorm:"not null"`
	Status        string    `gorm:"type:varchar(30);not null"`
	ActualOutput  *string   `gorm:"type:text"`
	ExecutionTime *int      `gorm:"type:int"`
	MemoryUsed    *int      `gorm:"type:int"`
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
	db := getDB(ctx, r.db)
	if err := db.Where("submission_id = ?", submissionID).
		Order("test_index ASC").
		Find(&daos).Error; err != nil {
		return nil, err
	}

	results := make([]*entity.SubmissionResult, 0, len(daos))
	for i := range daos {
		results = append(results, toSubmissionResultEntity(&daos[i]))
	}

	return results, nil
}

func (r *submissionResultRepository) DeleteBySubmissionID(ctx context.Context, submissionID int64) error {
	db := getDB(ctx, r.db)
	return db.Where("submission_id = ?", submissionID).Delete(&SubmissionResultDAO{}).Error
}

func (r *submissionResultRepository) ReplaceBySubmissionID(ctx context.Context, submissionID int64, results []*entity.SubmissionResult) error {
	db := getDB(ctx, r.db)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("submission_id = ?", submissionID).Delete(&SubmissionResultDAO{}).Error; err != nil {
			return err
		}

		if len(results) == 0 {
			return nil
		}

		daos := make([]SubmissionResultDAO, 0, len(results))
		for _, item := range results {
			daos = append(daos, SubmissionResultDAO{
				SubmissionID:  submissionID,
				TestIndex:     item.TestIndex,
				Status:        string(item.Status),
				ActualOutput:  item.ActualOutput,
				ExecutionTime: item.ExecutionTime,
				MemoryUsed:    item.MemoryUsed,
			})
		}

		return tx.Create(&daos).Error
	})
}

func toSubmissionResultEntity(dao *SubmissionResultDAO) *entity.SubmissionResult {
	return &entity.SubmissionResult{
		ID:            dao.ID,
		SubmissionID:  dao.SubmissionID,
		TestIndex:     dao.TestIndex,
		Status:        entity.ResultStatus(dao.Status),
		ActualOutput:  dao.ActualOutput,
		ExecutionTime: dao.ExecutionTime,
		MemoryUsed:    dao.MemoryUsed,
		CreatedAt:     dao.CreatedAt,
	}
}
