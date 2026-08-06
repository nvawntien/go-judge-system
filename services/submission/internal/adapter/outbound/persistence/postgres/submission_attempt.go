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

type SubmissionAttemptDAO struct {
	ID                int64         `gorm:"primaryKey;autoIncrement"`
	SubmissionID      int64         `gorm:"not null;index;index:idx_submission_attempt_created,priority:1;index:idx_submission_attempt_lookup,priority:1"`
	AttemptID         string        `gorm:"column:attempt_id;type:varchar(64);not null;uniqueIndex;index:idx_submission_attempt_lookup,priority:2"`
	TriggerType       string        `gorm:"type:varchar(40);not null;index"`
	TriggeredByUserID *string       `gorm:"type:varchar(100);index"`
	Status            string        `gorm:"type:varchar(30);not null;index"`
	TestcaseVersion   *int          `gorm:"type:int"`
	TestCount         *int          `gorm:"type:int"`
	DatasetChecksum   *string       `gorm:"type:varchar(128)"`
	CreatedAt         time.Time     `gorm:"autoCreateTime;index;index:idx_submission_attempt_created,priority:2"`
	UpdatedAt         time.Time     `gorm:"autoUpdateTime"`
	Submission        SubmissionDAO `gorm:"foreignKey:SubmissionID;references:ID;constraint:OnDelete:CASCADE"`
}

func (SubmissionAttemptDAO) TableName() string { return "submission_attempts" }

type submissionAttemptRepository struct {
	db *gorm.DB
}

func NewSubmissionAttemptRepository(db *gorm.DB) outbound.SubmissionAttemptRepository {
	db.AutoMigrate(&SubmissionAttemptDAO{})
	db.Migrator().AlterColumn(&SubmissionAttemptDAO{}, "TriggeredByUserID")
	return &submissionAttemptRepository{db: db}
}

func (r *submissionAttemptRepository) Create(ctx context.Context, attempt *entity.SubmissionAttempt) error {
	if attempt == nil {
		return fmt.Errorf("create submission attempt: attempt is nil")
	}
	db := getDB(ctx, r.db)
	dao := toSubmissionAttemptDAO(attempt)
	if err := db.Create(dao).Error; err != nil {
		return fmt.Errorf("create submission attempt %q: %w", attempt.AttemptID, err)
	}
	attempt.ID = dao.ID
	return nil
}

func (r *submissionAttemptRepository) GetByAttemptID(ctx context.Context, attemptID string) (*entity.SubmissionAttempt, error) {
	attemptID = strings.TrimSpace(attemptID)
	if attemptID == "" {
		return nil, domain.ErrSubmissionNotFound
	}

	var dao SubmissionAttemptDAO
	db := getDB(ctx, r.db)
	if err := db.Where("attempt_id = ?", attemptID).First(&dao).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrSubmissionNotFound
		}
		return nil, fmt.Errorf("get submission attempt %q: %w", attemptID, err)
	}
	return toSubmissionAttemptEntity(&dao), nil
}

func (r *submissionAttemptRepository) MarkCompleted(
	ctx context.Context,
	attemptID string,
	status entity.Status,
	testcaseVersion *int,
	testCount *int,
	datasetChecksum *string,
) error {
	attemptID = strings.TrimSpace(attemptID)
	if attemptID == "" {
		return fmt.Errorf("mark submission attempt completed: attempt ID is required")
	}

	updates := map[string]interface{}{
		"status":           string(status),
		"testcase_version": testcaseVersion,
		"test_count":       testCount,
		"dataset_checksum": datasetChecksum,
		"updated_at":       time.Now(),
	}
	db := getDB(ctx, r.db)
	result := db.Model(&SubmissionAttemptDAO{}).Where("attempt_id = ?", attemptID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("mark submission attempt %q completed: %w", attemptID, result.Error)
	}
	return nil
}

func toSubmissionAttemptDAO(a *entity.SubmissionAttempt) *SubmissionAttemptDAO {
	return &SubmissionAttemptDAO{
		ID:                a.ID,
		SubmissionID:      a.SubmissionID,
		AttemptID:         a.AttemptID,
		TriggerType:       string(a.TriggerType),
		TriggeredByUserID: a.TriggeredByUserID,
		Status:            string(a.Status),
		TestcaseVersion:   a.TestcaseVersion,
		TestCount:         a.TestCount,
		DatasetChecksum:   a.DatasetChecksum,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
	}
}

func toSubmissionAttemptEntity(dao *SubmissionAttemptDAO) *entity.SubmissionAttempt {
	return &entity.SubmissionAttempt{
		ID:                dao.ID,
		SubmissionID:      dao.SubmissionID,
		AttemptID:         dao.AttemptID,
		TriggerType:       entity.AttemptTriggerType(dao.TriggerType),
		TriggeredByUserID: dao.TriggeredByUserID,
		Status:            entity.Status(dao.Status),
		TestcaseVersion:   dao.TestcaseVersion,
		TestCount:         dao.TestCount,
		DatasetChecksum:   dao.DatasetChecksum,
		CreatedAt:         dao.CreatedAt,
		UpdatedAt:         dao.UpdatedAt,
	}
}
