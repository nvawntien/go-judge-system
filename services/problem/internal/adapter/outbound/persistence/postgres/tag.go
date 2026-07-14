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

type TagDAO struct {
	ID          uint           `gorm:"primaryKey;autoIncrement"`
	Name        string         `gorm:"not null;size:100"`
	Slug        string         `gorm:"uniqueIndex;not null;size:120"`
	Description *string        `gorm:"type:text"`
	IsActive    bool           `gorm:"not null;default:true;index"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (TagDAO) TableName() string { return "tags" }

type ProblemTagDAO struct {
	ProblemID int64      `gorm:"primaryKey;autoIncrement:false;uniqueIndex:idx_problem_tag"`
	TagID     uint       `gorm:"primaryKey;autoIncrement:false;uniqueIndex:idx_problem_tag"`
	Problem   ProblemDAO `gorm:"foreignKey:ProblemID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Tag       TagDAO     `gorm:"foreignKey:TagID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (ProblemTagDAO) TableName() string { return "problem_tags" }

type tagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) outbound.TagRepository {
	_ = db.AutoMigrate(&TagDAO{}, &ProblemTagDAO{})
	return &tagRepository{db: db}
}

func (r *tagRepository) Create(ctx context.Context, tag *entity.Tag) error {
	dao := toTagDAO(tag)
	if err := r.db.WithContext(ctx).Create(dao).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return domain.ErrTagAlreadyExists
		}
		return err
	}

	tag.ID = dao.ID
	tag.CreatedAt = dao.CreatedAt
	tag.UpdatedAt = dao.UpdatedAt
	return nil
}

func (r *tagRepository) GetByID(ctx context.Context, id uint) (*entity.Tag, error) {
	var dao TagDAO
	if err := r.db.WithContext(ctx).First(&dao, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTagNotFound
		}
		return nil, err
	}
	return toTagEntity(&dao), nil
}

func (r *tagRepository) GetBySlug(ctx context.Context, slug string) (*entity.Tag, error) {
	var dao TagDAO
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&dao).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTagNotFound
		}
		return nil, err
	}
	return toTagEntity(&dao), nil
}

func (r *tagRepository) Update(ctx context.Context, tag *entity.Tag) error {
	if err := r.db.WithContext(ctx).Model(&TagDAO{}).Where("id = ?", tag.ID).
		Updates(map[string]interface{}{
			"name":        tag.Name,
			"slug":        tag.Slug,
			"description": nullableString(tag.Description),
			"is_active":   tag.IsActive,
			"updated_at":  time.Now(),
		}).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return domain.ErrTagAlreadyExists
		}
		return err
	}
	return nil
}

func (r *tagRepository) Deactivate(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&TagDAO{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_active":  false,
			"updated_at": time.Now(),
		}).Error
}

func (r *tagRepository) ListActive(ctx context.Context) ([]*entity.Tag, error) {
	var daos []TagDAO
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("name ASC").Find(&daos).Error; err != nil {
		return nil, err
	}
	return toTagEntities(daos), nil
}

func (r *tagRepository) ListAll(ctx context.Context) ([]*entity.Tag, error) {
	var daos []TagDAO
	if err := r.db.WithContext(ctx).Order("name ASC").Find(&daos).Error; err != nil {
		return nil, err
	}
	return toTagEntities(daos), nil
}

func (r *tagRepository) ListByIDs(ctx context.Context, ids []uint, activeOnly bool) ([]*entity.Tag, error) {
	if len(ids) == 0 {
		return []*entity.Tag{}, nil
	}

	query := r.db.WithContext(ctx).Where("id IN ?", ids)
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	var daos []TagDAO
	if err := query.Order("id ASC").Find(&daos).Error; err != nil {
		return nil, err
	}
	return toTagEntities(daos), nil
}

func (r *tagRepository) CountPublishedProblemsByTagID(ctx context.Context, tagID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("problem_tags").
		Joins("JOIN problems ON problems.id = problem_tags.problem_id").
		Where("problem_tags.tag_id = ?", tagID).
		Where("problems.is_hidden = ?", false).
		Where("problems.deleted_at IS NULL").
		Count(&count).Error
	return count, err
}

func toTagDAO(tag *entity.Tag) *TagDAO {
	return &TagDAO{
		ID:          tag.ID,
		Name:        tag.Name,
		Slug:        tag.Slug,
		Description: nullableString(tag.Description),
		IsActive:    tag.IsActive,
		CreatedAt:   tag.CreatedAt,
		UpdatedAt:   tag.UpdatedAt,
	}
}

func toTagEntity(dao *TagDAO) *entity.Tag {
	return &entity.Tag{
		ID:          dao.ID,
		Name:        dao.Name,
		Slug:        dao.Slug,
		Description: derefString(dao.Description),
		IsActive:    dao.IsActive,
		CreatedAt:   dao.CreatedAt,
		UpdatedAt:   dao.UpdatedAt,
	}
}

func toTagEntities(daos []TagDAO) []*entity.Tag {
	items := make([]*entity.Tag, 0, len(daos))
	for _, dao := range daos {
		items = append(items, toTagEntity(&dao))
	}
	return items
}

type problemTagRow struct {
	ProblemID    int64
	ID           uint
	Name         string
	Slug         string
	Description  string
	IsActive     bool
	TagCreatedAt time.Time `gorm:"column:created_at"`
	TagUpdatedAt time.Time `gorm:"column:updated_at"`
}

func loadTagsByProblemIDs(ctx context.Context, db *gorm.DB, problemIDs []int64) (map[int64][]entity.Tag, error) {
	if len(problemIDs) == 0 {
		return map[int64][]entity.Tag{}, nil
	}

	var rows []problemTagRow
	if err := db.WithContext(ctx).
		Table("problem_tags").
		Select(
			"problem_tags.problem_id, tags.id, tags.name, tags.slug, COALESCE(tags.description, '') AS description, tags.is_active, tags.created_at, tags.updated_at",
		).
		Joins("JOIN tags ON tags.id = problem_tags.tag_id").
		Where("problem_tags.problem_id IN ?", problemIDs).
		Where("tags.deleted_at IS NULL").
		Order("tags.name ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[int64][]entity.Tag, len(problemIDs))
	for _, row := range rows {
		result[row.ProblemID] = append(result[row.ProblemID], entity.Tag{
			ID:          row.ID,
			Name:        row.Name,
			Slug:        row.Slug,
			Description: row.Description,
			IsActive:    row.IsActive,
			CreatedAt:   row.TagCreatedAt,
			UpdatedAt:   row.TagUpdatedAt,
		})
	}

	return result, nil
}

func syncProblemTags(tx *gorm.DB, problemID int64, tags []entity.Tag) error {
	if err := tx.Where("problem_id = ?", problemID).Delete(&ProblemTagDAO{}).Error; err != nil {
		return err
	}

	if len(tags) == 0 {
		return nil
	}

	rows := make([]ProblemTagDAO, 0, len(tags))
	seen := make(map[uint]struct{}, len(tags))
	for _, tag := range tags {
		if tag.ID == 0 {
			continue
		}
		if _, ok := seen[tag.ID]; ok {
			continue
		}
		seen[tag.ID] = struct{}{}
		rows = append(rows, ProblemTagDAO{
			ProblemID: problemID,
			TagID:     tag.ID,
		})
	}

	if len(rows) == 0 {
		return nil
	}

	return tx.Create(&rows).Error
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nullableString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
