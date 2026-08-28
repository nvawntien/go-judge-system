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

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// exampleJSON is an adapter-level struct with json tags for JSONB serialization.
// Domain entity.ProblemExample has no tags (clean domain).
type exampleJSON struct {
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output,omitempty"`
	LegacyOutput   string `json:"output,omitempty"`
	Explanation    string `json:"explanation,omitempty"`
}

// ExamplesJSON maps []entity.ProblemExample ↔ PostgreSQL JSONB column
type ExamplesJSON []entity.ProblemExample

func (e ExamplesJSON) Value() (driver.Value, error) {
	items := make([]exampleJSON, len(e))
	for i, ex := range e {
		items[i] = exampleJSON{Input: ex.Input, ExpectedOutput: ex.Output, Explanation: ex.Explanation}
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
		output := item.ExpectedOutput
		if output == "" {
			output = item.LegacyOutput
		}
		result[i] = entity.ProblemExample{Input: item.Input, Output: output, Explanation: item.Explanation}
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

type ConstraintsJSON []string

func (c ConstraintsJSON) Value() (driver.Value, error) {
	if c == nil {
		return "[]", nil
	}
	b, err := json.Marshal(c)
	return string(b), err
}

func (c *ConstraintsJSON) Scan(src interface{}) error {
	if src == nil {
		*c = ConstraintsJSON{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("ConstraintsJSON.Scan: unsupported type %T", src)
	}
	return json.Unmarshal(data, c)
}

type ProblemDAO struct {
	ID           int64           `gorm:"primaryKey;autoIncrement"`
	TitleSlug    string          `gorm:"column:title_slug;uniqueIndex;not null;size:500"`
	Title        string          `gorm:"not null;size:500"`
	Description  string          `gorm:"type:text;not null"`
	InputFormat  string          `gorm:"type:text;not null;default:''"`
	OutputFormat string          `gorm:"type:text;not null;default:''"`
	Difficulty   string          `gorm:"not null;size:20"`
	Examples     ExamplesJSON    `gorm:"type:jsonb;not null;default:'[]'"`
	Constraints  ConstraintsJSON `gorm:"type:jsonb;not null;default:'[]'"`
	Hints        HintsJSON       `gorm:"type:jsonb;not null;default:'[]'"`
	TimeLimit    float64         `gorm:"not null"`
	MemoryLimit  int             `gorm:"not null"`
	AuthorID     string          `gorm:"not null;size:100;index"`
	IsHidden     bool            `gorm:"default:true"`
	CreatedAt    time.Time       `gorm:"autoCreateTime"`
	UpdatedAt    time.Time       `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt  `gorm:"index"`
}

func (ProblemDAO) TableName() string { return "problems" }

// ── Repository ──────────────────────────────────────────────────────────────

type problemRepository struct{ db *gorm.DB }

type submissionProblemRepository struct{ db *gorm.DB }

func NewProblemRepository(db *gorm.DB) outbound.ProblemRepository {
	_ = db.AutoMigrate(&ProblemDAO{})
	return &problemRepository{db: db}
}

// NewSubmissionProblemRepository reads only the canonical fields needed by
// synchronous submission eligibility. It intentionally does not migrate or
// load tags/statement content.
func NewSubmissionProblemRepository(db *gorm.DB) outbound.SubmissionProblemRepository {
	return &submissionProblemRepository{db: db}
}

func (r *submissionProblemRepository) GetSubmissionProblem(
	ctx context.Context,
	id int64,
) (outbound.SubmissionProblemMetadata, error) {
	var row struct {
		ID          int64
		Title       string
		TitleSlug   string
		TimeLimit   float64
		MemoryLimit int
		AuthorID    string
		IsHidden    bool
	}

	err := r.db.WithContext(ctx).
		Model(&ProblemDAO{}).
		Select("id", "title", "title_slug", "time_limit", "memory_limit", "author_id", "is_hidden").
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return outbound.SubmissionProblemMetadata{}, domain.ErrProblemNotFound
		}
		return outbound.SubmissionProblemMetadata{}, err
	}

	return outbound.SubmissionProblemMetadata{
		ID:          row.ID,
		Title:       row.Title,
		Slug:        row.TitleSlug,
		TimeLimit:   row.TimeLimit,
		MemoryLimit: row.MemoryLimit,
		AuthorID:    row.AuthorID,
		IsHidden:    row.IsHidden,
	}, nil
}

func (r *problemRepository) Create(ctx context.Context, problem *entity.Problem) error {
	dao := toProblemDAO(problem)
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := tx.Create(dao).Error; err != nil {
		tx.Rollback()
		if isUniqueViolation(err) {
			return domain.ErrProblemAlreadyExists
		}
		return err
	}

	if err := syncProblemTags(tx, dao.ID, problem.Tags); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
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
	tagsByProblemID, err := loadTagsByProblemIDs(ctx, r.db, []int64{dao.ID})
	if err != nil {
		return nil, err
	}
	return toProblemEntity(&dao, tagsByProblemID[dao.ID]), nil
}

func (r *problemRepository) GetBySlug(ctx context.Context, slug string) (*entity.Problem, error) {
	var dao ProblemDAO
	if err := r.db.WithContext(ctx).Where("title_slug = ?", slug).First(&dao).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProblemNotFound
		}
		return nil, err
	}
	tagsByProblemID, err := loadTagsByProblemIDs(ctx, r.db, []int64{dao.ID})
	if err != nil {
		return nil, err
	}
	return toProblemEntity(&dao, tagsByProblemID[dao.ID]), nil
}

func (r *problemRepository) Update(ctx context.Context, problem *entity.Problem) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := tx.Model(&ProblemDAO{}).Where("id = ?", problem.ID).
		Updates(map[string]interface{}{
			"title_slug":    problem.TitleSlug,
			"title":         problem.Title,
			"description":   problem.Description,
			"input_format":  problem.InputFormat,
			"output_format": problem.OutputFormat,
			"difficulty":    normalizeDifficultyValue(problem.Difficulty),
			"examples":      ExamplesJSON(problem.Examples),
			"constraints":   ConstraintsJSON(problem.Constraints),
			"hints":         HintsJSON(problem.Hints),
			"time_limit":    problem.TimeLimit,
			"memory_limit":  problem.MemoryLimit,
			"is_hidden":     problem.IsHidden,
			"updated_at":    time.Now(),
		}).Error; err != nil {
		tx.Rollback()
		if isUniqueViolation(err) {
			return domain.ErrProblemAlreadyExists
		}
		return err
	}

	if err := syncProblemTags(tx, problem.ID, problem.Tags); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *problemRepository) Delete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&ProblemDAO{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrProblemNotFound
	}
	return nil
}

func (r *problemRepository) List(ctx context.Context, offset, limit int, difficulty, search, tagSlug string, includeHidden bool) ([]*entity.Problem, error) {
	db := r.db.WithContext(ctx)
	query := db.Model(&ProblemDAO{})
	if !includeHidden {
		query = query.Where("is_hidden = ?", false)
	}
	query = applyFilters(db, query, difficulty, search, tagSlug, !includeHidden)

	var daos []ProblemDAO
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&daos).Error; err != nil {
		return nil, err
	}

	tagsByProblemID, err := loadTagsByProblemIDs(ctx, r.db, collectProblemIDs(daos))
	if err != nil {
		return nil, err
	}

	return toProblemEntities(daos, tagsByProblemID), nil
}

func (r *problemRepository) Count(ctx context.Context, difficulty, search, tagSlug string, includeHidden bool) (int64, error) {
	db := r.db.WithContext(ctx)
	query := db.Model(&ProblemDAO{})
	if !includeHidden {
		query = query.Where("is_hidden = ?", false)
	}
	query = applyFilters(db, query, difficulty, search, tagSlug, !includeHidden)

	var count int64
	return count, query.Count(&count).Error
}

func (r *problemRepository) ListByAuthor(ctx context.Context, authorID string, offset, limit int, difficulty, search, tagSlug string) ([]*entity.Problem, error) {
	db := r.db.WithContext(ctx)
	query := db.Model(&ProblemDAO{}).Where("author_id = ?", authorID)
	query = applyFilters(db, query, difficulty, search, tagSlug, false)

	var daos []ProblemDAO
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&daos).Error; err != nil {
		return nil, err
	}

	tagsByProblemID, err := loadTagsByProblemIDs(ctx, r.db, collectProblemIDs(daos))
	if err != nil {
		return nil, err
	}

	return toProblemEntities(daos, tagsByProblemID), nil
}

func (r *problemRepository) CountByAuthor(ctx context.Context, authorID string, difficulty, search, tagSlug string) (int64, error) {
	db := r.db.WithContext(ctx)
	query := db.Model(&ProblemDAO{}).Where("author_id = ?", authorID)
	query = applyFilters(db, query, difficulty, search, tagSlug, false)

	var count int64
	return count, query.Count(&count).Error
}

func (r *problemRepository) ListForFormatBackfill(ctx context.Context) ([]*entity.Problem, error) {
	var daos []ProblemDAO
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&daos).Error; err != nil {
		return nil, err
	}

	tagsByProblemID, err := loadTagsByProblemIDs(ctx, r.db, collectProblemIDs(daos))
	if err != nil {
		return nil, err
	}
	return toProblemEntities(daos, tagsByProblemID), nil
}

func (r *problemRepository) UpdateFormatsForBackfill(ctx context.Context, id int64, description, inputFormat, outputFormat string) (bool, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return false, tx.Error
	}

	result := tx.Model(&ProblemDAO{}).
		Where("id = ? AND input_format = '' AND output_format = ''", id).
		Updates(map[string]interface{}{
			"description":   description,
			"input_format":  inputFormat,
			"output_format": outputFormat,
			"updated_at":    time.Now(),
		})
	if result.Error != nil {
		tx.Rollback()
		return false, result.Error
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return false, nil
	}
	if err := tx.Commit().Error; err != nil {
		return false, err
	}
	return true, nil
}

func applyFilters(db *gorm.DB, query *gorm.DB, difficulty, search, tagSlug string, requireActiveTag bool) *gorm.DB {
	difficulty = strings.ToLower(strings.TrimSpace(difficulty))
	search = strings.TrimSpace(search)
	tagSlug = strings.ToLower(strings.TrimSpace(tagSlug))

	if difficulty != "" {
		query = query.Where("difficulty = ?", difficulty)
	}
	if search != "" {
		like := "%" + escapeLikePattern(search) + "%"
		query = query.Where("(title ILIKE ? ESCAPE '\\' OR title_slug ILIKE ? ESCAPE '\\')", like, like)
	}
	if tagSlug != "" {
		query = query.Where("id IN (?)", filterProblemIDsByTagSlug(db, tagSlug, requireActiveTag))
	}
	return query
}

func filterProblemIDsByTagSlug(db *gorm.DB, tagSlug string, requireActiveTag bool) *gorm.DB {
	subQuery := db.
		Table("problem_tags").
		Select("problem_tags.problem_id").
		Joins("JOIN tags ON tags.id = problem_tags.tag_id").
		Where("tags.slug = ?", tagSlug).
		Where("tags.deleted_at IS NULL")

	if requireActiveTag {
		subQuery = subQuery.Where("tags.is_active = ?", true)
	}

	return subQuery
}

func toProblemDAO(p *entity.Problem) *ProblemDAO {
	return &ProblemDAO{
		ID:           p.ID,
		TitleSlug:    p.TitleSlug,
		Title:        p.Title,
		Description:  p.Description,
		InputFormat:  p.InputFormat,
		OutputFormat: p.OutputFormat,
		Difficulty:   normalizeDifficultyValue(p.Difficulty),
		Examples:     ExamplesJSON(p.Examples),
		Constraints:  ConstraintsJSON(p.Constraints),
		Hints:        HintsJSON(p.Hints),
		TimeLimit:    p.TimeLimit,
		MemoryLimit:  p.MemoryLimit,
		AuthorID:     p.AuthorID,
		IsHidden:     p.IsHidden,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func toProblemEntity(dao *ProblemDAO, tags []entity.Tag) *entity.Problem {
	p := &entity.Problem{
		ID:           dao.ID,
		TitleSlug:    dao.TitleSlug,
		Title:        dao.Title,
		Description:  dao.Description,
		InputFormat:  dao.InputFormat,
		OutputFormat: dao.OutputFormat,
		Difficulty:   entity.Difficulty(dao.Difficulty),
		Tags:         tags,
		Examples:     []entity.ProblemExample(dao.Examples),
		Constraints:  []string(dao.Constraints),
		Hints:        []string(dao.Hints),
		TimeLimit:    dao.TimeLimit,
		MemoryLimit:  dao.MemoryLimit,
		AuthorID:     dao.AuthorID,
		IsHidden:     dao.IsHidden,
		CreatedAt:    dao.CreatedAt,
		UpdatedAt:    dao.UpdatedAt,
	}
	if dao.DeletedAt.Valid {
		t := dao.DeletedAt.Time
		p.DeletedAt = &t
	}
	return p
}

func toProblemEntities(daos []ProblemDAO, tagsByProblemID map[int64][]entity.Tag) []*entity.Problem {
	results := make([]*entity.Problem, 0, len(daos))
	for _, dao := range daos {
		results = append(results, toProblemEntity(&dao, tagsByProblemID[dao.ID]))
	}
	return results
}

func collectProblemIDs(daos []ProblemDAO) []int64 {
	ids := make([]int64, 0, len(daos))
	for _, dao := range daos {
		ids = append(ids, dao.ID)
	}
	return ids
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func normalizeDifficultyValue(difficulty entity.Difficulty) string {
	return strings.ToLower(strings.TrimSpace(string(difficulty)))
}

func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
