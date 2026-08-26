package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"

	"gorm.io/gorm"
)

type UserDAO struct {
	ID              string    `gorm:"primaryKey;type:uuid"`
	FullName        string    `gorm:"size:255"`
	Username        string    `gorm:"uniqueIndex;not null;size:100"`
	Email           string    `gorm:"uniqueIndex;not null;size:255"`
	Password        string    `gorm:"not null"`
	Role            rbac.Role `gorm:"default:'user';size:20"`
	Rating          int       `gorm:"default:0"`
	IsActive        bool      `gorm:"default:false"`
	IsSuspended     bool      `gorm:"default:false"`
	AvatarURL       *string   `gorm:"size:500"`
	AvatarObjectKey *string   `gorm:"column:avatar_object_key;type:text"`
	Bio             *string   `gorm:"size:500"`
	Country         *string   `gorm:"size:100"`
	School          *string   `gorm:"size:255"`
	Company         *string   `gorm:"size:255"`
	GithubURL       *string   `gorm:"size:500"`
	WebsiteURL      *string   `gorm:"size:500"`
	LinkedinURL     *string   `gorm:"size:500"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (UserDAO) TableName() string {
	return "users"
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) outbound.UserRepository {
	db.AutoMigrate(&UserDAO{})
	return newUserRepository(db)
}

// NewUserRepositoryForManagedSchema constructs the normal user repository
// without changing the database schema. Operator commands use it after the
// deployed Auth service has already managed the schema.
func NewUserRepositoryForManagedSchema(db *gorm.DB) outbound.UserRepository {
	return newUserRepository(db)
}

func newUserRepository(db *gorm.DB) outbound.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(ctx context.Context, user *entity.User) error {
	if err := r.db.WithContext(ctx).Create(toUserDAO(user)).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrDuplicateEntry
		}
		return err
	}
	return nil
}

func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&UserDAO{}).Error
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var dao UserDAO
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&dao).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return toUserEntity(&dao), nil
}

func (r *userRepository) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
	var dao UserDAO
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&dao).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return toUserEntity(&dao), nil
}

func (r *userRepository) GetUserById(ctx context.Context, id string) (*entity.User, error) {
	var dao UserDAO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&dao).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return toUserEntity(&dao), nil
}

func (r *userRepository) ListUsers(ctx context.Context, filter outbound.ListUsersFilter) (outbound.ListUsersResult, error) {
	query := r.db.WithContext(ctx).Model(&UserDAO{})
	search := strings.TrimSpace(filter.Search)
	if search != "" {
		pattern := "%" + escapeLikePattern(search) + "%"
		query = query.Where("(username ILIKE ? ESCAPE '\\' OR email ILIKE ? ESCAPE '\\' OR full_name ILIKE ? ESCAPE '\\')", pattern, pattern, pattern)
	}
	if filter.Role != nil {
		query = query.Where("role = ?", *filter.Role)
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}
	if filter.IsSuspended != nil {
		query = query.Where("is_suspended = ?", *filter.IsSuspended)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return outbound.ListUsersResult{}, err
	}

	var daos []UserDAO
	if err := query.Order("created_at DESC").Order("id DESC").Offset(filter.Offset).Limit(filter.Limit).Find(&daos).Error; err != nil {
		return outbound.ListUsersResult{}, err
	}

	items := make([]*entity.User, 0, len(daos))
	for i := range daos {
		items = append(items, toUserEntity(&daos[i]))
	}
	return outbound.ListUsersResult{Items: items, Total: total}, nil
}

// SearchPublicUsers searches only profile-visible accounts and identity fields.
// Keep this separate from ListUsers: admin search intentionally includes email
// and account-state filters that must never become public discovery behavior.
func (r *userRepository) SearchPublicUsers(ctx context.Context, filter outbound.SearchPublicUsersFilter) (outbound.SearchPublicUsersResult, error) {
	pattern := "%" + escapeLikePattern(strings.TrimSpace(filter.Query)) + "%"
	query := r.db.WithContext(ctx).Model(&UserDAO{}).
		Where("is_active = ? AND is_suspended = ?", true, false).
		Where("(username ILIKE ? ESCAPE '\\' OR full_name ILIKE ? ESCAPE '\\')", pattern, pattern)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return outbound.SearchPublicUsersResult{}, err
	}

	var daos []UserDAO
	if err := query.Order("username ASC").Order("id ASC").Offset(filter.Offset).Limit(filter.Limit).Find(&daos).Error; err != nil {
		return outbound.SearchPublicUsersResult{}, err
	}

	items := make([]*entity.User, 0, len(daos))
	for i := range daos {
		items = append(items, toUserEntity(&daos[i]))
	}
	return outbound.SearchPublicUsersResult{Items: items, Total: total}, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Model(&UserDAO{}).
		Where("id = ?", user.ID).
		Updates(map[string]interface{}{
			"full_name":         user.FullName,
			"username":          user.Username,
			"email":             user.Email,
			"password":          user.Password,
			"role":              user.Role,
			"rating":            user.Rating,
			"is_active":         user.IsActive,
			"is_suspended":      user.IsSuspended,
			"avatar_url":        user.AvatarURL,
			"avatar_object_key": user.AvatarObjectKey,
			"bio":               user.Bio,
			"country":           user.Country,
			"school":            user.School,
			"company":           user.Company,
			"github_url":        user.GithubURL,
			"website_url":       user.WebsiteURL,
			"linkedin_url":      user.LinkedinURL,
			"updated_at":        user.UpdatedAt,
		}).Error
}

func (r *userRepository) UpdatePassword(ctx context.Context, userID string, passwordHash string, updatedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&UserDAO{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password":   passwordHash,
			"updated_at": updatedAt,
		}).Error
}

// RotateBenchmarkPasswords updates only the password column in one
// transaction. Each update repeats the canonical fixture predicates so a
// concurrent role/status/identity change rolls back the entire rotation.
func (r *userRepository) RotateBenchmarkPasswords(ctx context.Context, updates []outbound.BenchmarkPasswordUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			result := tx.Model(&UserDAO{}).
				Where("id = ?", update.UserID).
				Where("username = ?", update.Username).
				Where("email = ?", update.Email).
				Where("full_name = ?", update.FullName).
				Where("role = ?", rbac.RoleUser).
				Where("is_active = ?", true).
				Where("is_suspended = ?", false).
				UpdateColumn("password", update.PasswordHash)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("canonical benchmark identity changed during password rotation")
			}
		}
		return nil
	})
}

func (r *userRepository) UpdateProfile(ctx context.Context, userID string, updates outbound.ProfileUpdates) error {
	values := map[string]interface{}{"updated_at": updates.UpdatedAt}
	if updates.FullName != nil {
		values["full_name"] = *updates.FullName
	}
	if updates.Bio != nil {
		values["bio"] = updates.Bio
	}
	if updates.Country != nil {
		values["country"] = updates.Country
	}
	if updates.School != nil {
		values["school"] = updates.School
	}
	if updates.Company != nil {
		values["company"] = updates.Company
	}
	if updates.GithubURL != nil {
		values["github_url"] = updates.GithubURL
	}
	if updates.WebsiteURL != nil {
		values["website_url"] = updates.WebsiteURL
	}
	if updates.LinkedinURL != nil {
		values["linkedin_url"] = updates.LinkedinURL
	}

	return r.db.WithContext(ctx).Model(&UserDAO{}).Where("id = ?", userID).Updates(values).Error
}

func (r *userRepository) UpdateAvatar(ctx context.Context, userID string, avatarURL string, avatarObjectKey string, updatedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&UserDAO{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"avatar_url":        avatarURL,
			"avatar_object_key": avatarObjectKey,
			"updated_at":        updatedAt,
		}).Error
}

func escapeLikePattern(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "%", "\\%")
	return strings.ReplaceAll(value, "_", "\\_")
}

// mapping entity to dao
func toUserDAO(user *entity.User) *UserDAO {
	return &UserDAO{
		ID:              user.ID,
		FullName:        user.FullName,
		Username:        user.Username,
		Email:           user.Email,
		Password:        user.Password,
		Role:            user.Role,
		Rating:          user.Rating,
		IsActive:        user.IsActive,
		IsSuspended:     user.IsSuspended,
		AvatarURL:       user.AvatarURL,
		AvatarObjectKey: user.AvatarObjectKey,
		Bio:             user.Bio,
		Country:         user.Country,
		School:          user.School,
		Company:         user.Company,
		GithubURL:       user.GithubURL,
		WebsiteURL:      user.WebsiteURL,
		LinkedinURL:     user.LinkedinURL,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
}

// mapping dao to entity
func toUserEntity(dao *UserDAO) *entity.User {
	return &entity.User{
		ID:              dao.ID,
		FullName:        dao.FullName,
		Username:        dao.Username,
		Email:           dao.Email,
		Password:        dao.Password,
		Role:            dao.Role,
		Rating:          dao.Rating,
		IsActive:        dao.IsActive,
		IsSuspended:     dao.IsSuspended,
		AvatarURL:       dao.AvatarURL,
		AvatarObjectKey: dao.AvatarObjectKey,
		Bio:             dao.Bio,
		Country:         dao.Country,
		School:          dao.School,
		Company:         dao.Company,
		GithubURL:       dao.GithubURL,
		WebsiteURL:      dao.WebsiteURL,
		LinkedinURL:     dao.LinkedinURL,
		CreatedAt:       dao.CreatedAt,
		UpdatedAt:       dao.UpdatedAt,
	}
}
