package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"

	"gorm.io/gorm"
)

type ProblemDAO struct {
	ID          int64          `gorm:"primaryKey;autoIncrement"`
	Slug        string         `gorm:"uniqueIndex;not null;size:500"`
	Title       string         `gorm:"not null;size:500"`
	Description string         `gorm:"type:text;not null"`
	Difficulty  string         `gorm:"not null;size:20"`
	TimeLimit   int            `gorm:"not null"`
	MemoryLimit int            `gorm:"not null"`
	AuthorID    string         `gorm:"not null;size:100;index"`
	IsHidden    bool           `gorm:"default:true"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (ProblemDAO) TableName() string { return "problems" }

type problemRepository struct{ db *gorm.DB }

func NewProblemRepository(db *gorm.DB) outbound.ProblemRepository {
	db.AutoMigrate(&ProblemDAO{})
	return &problemRepository{db: db}
}

func (r *problemRepository) Create(ctx context.Context, problem *entity.Problem) error {
	dao := toProblemDAO(problem)
	if err := r.db.WithContext(ctx).Create(dao).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return domain.ErrProblemAlreadyExists
		}
		return err
	}
	problem.ID = dao.ID
	return nil
}

func (r *problemRepository) GetByID(ctx context.Context, id int64) (*entity.Problem, error) {
	var dao ProblemDAO
	if err := r.db.WithContext(ctx).First(&dao, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProblemNotFound
		}
		return nil, err
	}
	return toProblemEntity(&dao), nil
}

func (r *problemRepository) GetBySlug(ctx context.Context, slug string) (*entity.Problem, error) {
	var dao ProblemDAO
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&dao).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProblemNotFound
		}
		return nil, err
	}
	return toProblemEntity(&dao), nil
}

func (r *problemRepository) Update(ctx context.Context, problem *entity.Problem) error {
	return r.db.WithContext(ctx).Model(&ProblemDAO{}).Where("id = ?", problem.ID).
		Updates(map[string]interface{}{
			"slug":        problem.Slug,
			"title":       problem.Title,
			"description": problem.Description,
			"difficulty":  string(problem.Difficulty),
			"time_limit":  problem.TimeLimit,
			"memory_limit": problem.MemoryLimit,
			"is_hidden":   problem.IsHidden,
			"updated_at":  time.Now(),
		}).Error
}

func (r *problemRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&ProblemDAO{}, id).Error // GORM soft delete
}

func (r *problemRepository) List(ctx context.Context, offset, limit int, difficulty, search string, includeHidden bool) ([]*entity.Problem, error) {
	query := r.db.WithContext(ctx)
	if !includeHidden {
		query = query.Where("is_hidden = ?", false)
	}
	query = applyFilters(query, difficulty, search)

	var daos []ProblemDAO
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&daos).Error; err != nil {
		return nil, err
	}
	return toProblemEntities(daos), nil
}

func (r *problemRepository) Count(ctx context.Context, difficulty, search string, includeHidden bool) (int64, error) {
	query := r.db.WithContext(ctx).Model(&ProblemDAO{})
	if !includeHidden {
		query = query.Where("is_hidden = ?", false)
	}
	query = applyFilters(query, difficulty, search)

	var count int64
	return count, query.Count(&count).Error
}

func (r *problemRepository) ListByAuthor(ctx context.Context, authorID string, offset, limit int, difficulty, search string) ([]*entity.Problem, error) {
	query := r.db.WithContext(ctx).Where("author_id = ?", authorID)
	query = applyFilters(query, difficulty, search)

	var daos []ProblemDAO
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&daos).Error; err != nil {
		return nil, err
	}
	return toProblemEntities(daos), nil
}

func (r *problemRepository) CountByAuthor(ctx context.Context, authorID string, difficulty, search string) (int64, error) {
	query := r.db.WithContext(ctx).Model(&ProblemDAO{}).Where("author_id = ?", authorID)
	query = applyFilters(query, difficulty, search)

	var count int64
	return count, query.Count(&count).Error
}

// ---- Helpers ----

func applyFilters(query *gorm.DB, difficulty, search string) *gorm.DB {
	if difficulty != "" {
		query = query.Where("difficulty = ?", difficulty)
	}
	if search != "" {
		query = query.Where("title ILIKE ?", "%"+search+"%")
	}
	return query
}

func toProblemDAO(p *entity.Problem) *ProblemDAO {
	return &ProblemDAO{
		ID: p.ID, Slug: p.Slug, Title: p.Title, Description: p.Description,
		Difficulty: string(p.Difficulty), TimeLimit: p.TimeLimit, MemoryLimit: p.MemoryLimit,
		AuthorID: p.AuthorID, IsHidden: p.IsHidden, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toProblemEntity(dao *ProblemDAO) *entity.Problem {
	p := &entity.Problem{
		ID: dao.ID, Slug: dao.Slug, Title: dao.Title, Description: dao.Description,
		Difficulty: entity.Difficulty(dao.Difficulty), TimeLimit: dao.TimeLimit, MemoryLimit: dao.MemoryLimit,
		AuthorID: dao.AuthorID, IsHidden: dao.IsHidden, CreatedAt: dao.CreatedAt, UpdatedAt: dao.UpdatedAt,
	}
	if dao.DeletedAt.Valid {
		t := dao.DeletedAt.Time
		p.DeletedAt = &t
	}
	return p
}

func toProblemEntities(daos []ProblemDAO) []*entity.Problem {
	results := make([]*entity.Problem, 0, len(daos))
	for _, dao := range daos {
		results = append(results, toProblemEntity(&dao))
	}
	return results
}