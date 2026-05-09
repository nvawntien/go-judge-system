package entity

import (
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/domain/valueobject"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        string
	FullName  string
	Username  string
	Email     string
	Password  string
	Role      rbac.Role
	Rating    int
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewUser(fullName string, username string, email valueobject.Email, password valueobject.Password) *User {
	return &User{
		ID:        uuid.New().String(),
		FullName:  fullName,
		Username:  username,
		Email:     email.String(),
		Password:  password.Hash(),
		Role:      rbac.RoleUser,
		Rating:    0,
		IsActive:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (u *User) Activate() {
	u.IsActive = true
	u.UpdatedAt = time.Now()
}

func (u *User) UpdatePassword(newPassword valueobject.Password) {
	u.Password = newPassword.Hash()
	u.UpdatedAt = time.Now()
}

func (u *User) AssignRole(role rbac.Role) {
	u.Role = role
	u.UpdatedAt = time.Now()
}
