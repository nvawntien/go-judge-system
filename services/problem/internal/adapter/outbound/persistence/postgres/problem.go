package postgres

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"

	"gorm.io/gorm"
)

// ── JSONB custom types for GORM ─────────────────────────────────────────────

// exampleJSON is an adapter-level struct with json tags for JSONB serialization.
// Domain entity.ProblemExample has no tags (clean domain).
type exampleJSON struct {
	Input       string `json:"input"`
	Output      string `json:"output"`
	Explanation string `json:"explanation,omitempty"`
}

// ExamplesJSON maps []entity.ProblemExample ↔ PostgreSQL JSONB column
type ExamplesJSON []entity.ProblemExample

func (e ExamplesJSON) Value() (driver.Value, error) {
	items := make([]exampleJSON, len(e))
	for i, ex := range e {
		items[i] = exampleJSON{Input: ex.Input, Output: ex.Output, Explanation: ex.Explanation}
	}
	b, err := json.Marshal(items)
	return string(b), err
}

func (e *ExamplesJSON) Scan(src interface{}) error {
	if src == nil {
		*e = ExamplesJSON{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("ExamplesJSON.Scan: unsupported type %T", src)
	}
	var items []exampleJSON
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	result := make(ExamplesJSON, len(items))
	for i, item := range items {
		result[i] = entity.ProblemExample{Input: item.Input, Output: item.Output, Explanation: item.Explanation}
	}
	*e = result
	return nil
}

// HintsJSON maps []string ↔ PostgreSQL JSONB column
type HintsJSON []string

func (h HintsJSON) Value() (driver.Value, error) {
	if h == nil {
		return "[]", nil
	}
	b, err := json.Marshal(h)
	return string(b), err
}

func (h *HintsJSON) Scan(src interface{}) error {
	if src == nil {
		*h = HintsJSON{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("HintsJSON.Scan: unsupported type %T", src)
	}
	return json.Unmarshal(data, h)
}

// ── DAO ─────────────────────────────────────────────────────────────────────

type ProblemDAO struct {
	ID          int64          `gorm:"primaryKey;autoIncrement"`
	TitleSlug   string         `gorm:"column:title_slug;uniqueIndex;not null;size:500"`
	Title       string         `gorm:"not null;size:500"`
	Description string         `gorm:"type:text;not null"`
	Difficulty  string         `gorm:"not null;size:20"`
	Examples    ExamplesJSON   `gorm:"type:jsonb;not null;default:'[]'"`
	Constraints string         `gorm:"type:text;not null;default:''"`
	Hints       HintsJSON      `gorm:"type:jsonb;not null;default:'[]'"`
	TimeLimit   float64        `gorm:"not null"`
	MemoryLimit int            `gorm:"not null"`
	AuthorID    string         `gorm:"not null;size:100;index"`
	IsHidden    bool           `gorm:"default:true"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (ProblemDAO) TableName() string { return "problems" }

// ── Repository ──────────────────────────────────────────────────────────────

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
	if err := r.db.WithContext(ctx).Where("title_slug = ?", slug).First(&dao).Error; err != nil {
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
			"title_slug":  problem.TitleSlug,
			"title":       problem.Title,
			"description": problem.Description,
			"difficulty":  string(problem.Difficulty),
			"examples":    ExamplesJSON(problem.Examples),
			"constraints": problem.Constraints,
			"hints":       HintsJSON(problem.Hints),
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
		ID:          p.ID,
		TitleSlug:   p.TitleSlug,
		Title:       p.Title,
		Description: p.Description,
		Difficulty:  string(p.Difficulty),
		Examples:    ExamplesJSON(p.Examples),
		Constraints: p.Constraints,
		Hints:       HintsJSON(p.Hints),
		TimeLimit:   p.TimeLimit,
		MemoryLimit: p.MemoryLimit,
		AuthorID:    p.AuthorID,
		IsHidden:    p.IsHidden,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func toProblemEntity(dao *ProblemDAO) *entity.Problem {
	p := &entity.Problem{
		ID:          dao.ID,
		TitleSlug:   dao.TitleSlug,
		Title:       dao.Title,
		Description: dao.Description,
		Difficulty:  entity.Difficulty(dao.Difficulty),
		Examples:    []entity.ProblemExample(dao.Examples),
		Constraints: dao.Constraints,
		Hints:       []string(dao.Hints),
		TimeLimit:   dao.TimeLimit,
		MemoryLimit: dao.MemoryLimit,
		AuthorID:    dao.AuthorID,
		IsHidden:    dao.IsHidden,
		CreatedAt:   dao.CreatedAt,
		UpdatedAt:   dao.UpdatedAt,
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