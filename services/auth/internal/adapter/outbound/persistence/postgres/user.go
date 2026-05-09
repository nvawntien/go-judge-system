package postgres

import (
	"context"
	"errors"
	"time"

	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"

	"gorm.io/gorm"
)

type UserDAO struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	FullName  string    `gorm:"size:255"`
	Username  string    `gorm:"uniqueIndex;not null;size:100"`
	Email     string    `gorm:"uniqueIndex;not null;size:255"`
	Password  string    `gorm:"not null"`
	Role      rbac.Role `gorm:"default:'user';size:20"`
	Rating    int       `gorm:"default:0"`
	IsActive  bool      `gorm:"default:false"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (UserDAO) TableName() string {
	return "users"
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) outbound.UserRepository {
	db.AutoMigrate(&UserDAO{})
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

func (r *userRepository) UpdateUser(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Model(&UserDAO{}).
		Where("id = ?", user.ID).
		Updates(map[string]interface{}{
			"full_name":  user.FullName,
			"username":   user.Username,
			"email":      user.Email,
			"password":   user.Password,
			"role":       user.Role,
			"rating":     user.Rating,
			"is_active":  user.IsActive,
			"updated_at": user.UpdatedAt,
		}).Error
}

// mapping entity to dao
func toUserDAO(user *entity.User) *UserDAO {
	return &UserDAO{
		ID:        user.ID,
		FullName:  user.FullName,
		Username:  user.Username,
		Email:     user.Email,
		Password:  user.Password,
		Role:      user.Role,
		Rating:    user.Rating,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// mapping dao to entity
func toUserEntity(dao *UserDAO) *entity.User {
	return &entity.User{
		ID:        dao.ID,
		FullName:  dao.FullName,
		Username:  dao.Username,
		Email:     dao.Email,
		Password:  dao.Password,
		Role:      dao.Role,
		Rating:    dao.Rating,
		IsActive:  dao.IsActive,
		CreatedAt: dao.CreatedAt,
		UpdatedAt: dao.UpdatedAt,
	}
}
