package postgres

import (
	"context"
	"errors"
	"time"

	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TestCaseDAO struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	ProblemID    int64     `gorm:"uniqueIndex;not null"`
	ZipObjectKey string    `gorm:"type:varchar(500);not null"`
	TestCount    int       `gorm:"not null"`
	Version      string    `gorm:"type:varchar(50);not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

func (TestCaseDAO) TableName() string { return "test_cases" }

type testCaseRepository struct {
	db *gorm.DB
}

func NewTestCaseRepository(db *gorm.DB) outbound.TestCaseRepository {
	db.AutoMigrate(&TestCaseDAO{})
	return &testCaseRepository{db: db}
}

func (r *testCaseRepository) Upsert(ctx context.Context, tc *entity.TestCase) error {
	dao := toTestCaseDAO(tc)
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "problem_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"zip_object_key",
			"test_count",
			"version",
		}),
	}).Create(dao).Error

	if err == nil {
		tc.ID = dao.ID
	}
	return err
}

func (r *testCaseRepository) GetByProblemID(ctx context.Context, problemID int64) (*entity.TestCase, error) {
	var dao TestCaseDAO
	if err := r.db.WithContext(ctx).Where("problem_id = ?", problemID).First(&dao).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTestCaseNotFound
		}
		return nil, err
	}
	return toTestCaseEntity(&dao), nil
}

func (r *testCaseRepository) DeleteByProblemID(ctx context.Context, problemID int64) error {
	return r.db.WithContext(ctx).Where("problem_id = ?", problemID).Delete(&TestCaseDAO{}).Error
}

func toTestCaseDAO(tc *entity.TestCase) *TestCaseDAO {
	return &TestCaseDAO{
		ID:           tc.ID,
		ProblemID:    tc.ProblemID,
		ZipObjectKey: tc.ZipObjectKey,
		TestCount:    tc.TestCount,
		Version:      tc.Version,
		CreatedAt:    tc.CreatedAt,
	}
}

func toTestCaseEntity(dao *TestCaseDAO) *entity.TestCase {
	return &entity.TestCase{
		ID:           dao.ID,
		ProblemID:    dao.ProblemID,
		ZipObjectKey: dao.ZipObjectKey,
		TestCount:    dao.TestCount,
		Version:      dao.Version,
		CreatedAt:    dao.CreatedAt,
	}
}