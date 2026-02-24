package postgres

import (
	"context"
	"errors"
	"time"

	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
	"go-judge-system/services/auth/internal/domain/valueobject"

	"gorm.io/gorm"
)

type UserDAO struct {
	ID        string    `gorm:"primaryKey;type:uuid"`
	Username  string    `gorm:"uniqueIndex;not null;size:100"`
	Email     string    `gorm:"uniqueIndex;not null;size:255"`
	Password  string    `gorm:"not null"`
	Role      string    `gorm:"default:'user';size:20"`
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
	dao := &UserDAO{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email.String(),
		Password:  user.Password,
		Role:      user.Role,
		Rating:    user.Rating,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	return r.db.WithContext(ctx).Create(dao).Error
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var dao UserDAO
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&dao).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	emailVO, _ := valueobject.NewEmail(dao.Email)

	return &entity.User{
		ID:        dao.ID,
		Username:  dao.Username,
		Email:     emailVO,
		Password:  dao.Password,
		Role:      dao.Role,
		Rating:    dao.Rating,
		IsActive:  dao.IsActive,
		CreatedAt: dao.CreatedAt,
		UpdatedAt: dao.UpdatedAt,
	}, nil
}