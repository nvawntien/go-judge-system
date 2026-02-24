package entity

import (
	"go-judge-system/services/auth/internal/domain/valueobject"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        string
	Username  string
	Email     valueobject.Email
	Password  string
	Role      string
	Rating    int
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewUser(username string, email valueobject.Email, password valueobject.Password) *User {
	return &User{
		ID:        uuid.New().String(),
		Username:  username,
		Email:     email,
		Password:  password.Hash(),
		Role:      "user",
		Rating:    0,
		IsActive:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}