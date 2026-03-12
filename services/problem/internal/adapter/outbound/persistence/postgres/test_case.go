package postgres

import (
	"context"
	"errors"
	"time"

	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"

	"gorm.io/gorm"
)

type TestCaseDAO struct {
	ID             int64     `gorm:"primaryKey;autoIncrement"`
	ProblemID      int64     `gorm:"not null;index"`
	Input          string    `gorm:"type:text;not null"`
	ExpectedOutput string    `gorm:"type:text;not null"`
	IsExample      bool      `gorm:"default:false"`
	Order          int       `gorm:"not null"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

func (TestCaseDAO) TableName() string { return "test_cases" }

type testCaseRepository struct{ db *gorm.DB }

func NewTestCaseRepository(db *gorm.DB) outbound.TestCaseRepository {
	db.AutoMigrate(&TestCaseDAO{})
	return &testCaseRepository{db: db}
}

func (r *testCaseRepository) Create(ctx context.Context, tc *entity.TestCase) error {
	dao := toTestCaseDAO(tc)
	if err := r.db.WithContext(ctx).Create(dao).Error; err != nil {
		return err
	}
	tc.ID = dao.ID
	return nil
}

func (r *testCaseRepository) GetByID(ctx context.Context, id int64) (*entity.TestCase, error) {
	var dao TestCaseDAO
	if err := r.db.WithContext(ctx).First(&dao, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTestCaseNotFound
		}
		return nil, err
	}
	return toTestCaseEntity(&dao), nil
}

func (r *testCaseRepository) Update(ctx context.Context, tc *entity.TestCase) error {
	return r.db.WithContext(ctx).Model(&TestCaseDAO{}).Where("id = ?", tc.ID).
		Updates(map[string]interface{}{
			"input":           tc.Input,
			"expected_output": tc.ExpectedOutput,
			"is_example":      tc.IsExample,
			"order":           tc.Order,
		}).Error
}

func (r *testCaseRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&TestCaseDAO{}, id).Error
}

func (r *testCaseRepository) GetByProblemID(ctx context.Context, problemID int64) ([]*entity.TestCase, error) {
	var daos []TestCaseDAO
	if err := r.db.WithContext(ctx).Where("problem_id = ?", problemID).Order(`"order" ASC`).Find(&daos).Error; err != nil {
		return nil, err
	}
	results := make([]*entity.TestCase, 0, len(daos))
	for _, dao := range daos {
		results = append(results, toTestCaseEntity(&dao))
	}
	return results, nil
}

func (r *testCaseRepository) CountByProblemID(ctx context.Context, problemID int64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&TestCaseDAO{}).Where("problem_id = ?", problemID).Count(&count).Error
	return int(count), err
}

func toTestCaseDAO(tc *entity.TestCase) *TestCaseDAO {
	return &TestCaseDAO{
		ID: tc.ID, ProblemID: tc.ProblemID, Input: tc.Input,
		ExpectedOutput: tc.ExpectedOutput, IsExample: tc.IsExample, Order: tc.Order,
		CreatedAt: tc.CreatedAt,
	}
}

func toTestCaseEntity(dao *TestCaseDAO) *entity.TestCase {
	return &entity.TestCase{
		ID: dao.ID, ProblemID: dao.ProblemID, Input: dao.Input,
		ExpectedOutput: dao.ExpectedOutput, IsExample: dao.IsExample, Order: dao.Order,
		CreatedAt: dao.CreatedAt,
	}
}
