package entity

import (
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/domain/valueobject"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              string
	FullName        string
	Username        string
	Email           string
	Password        string
	Role            rbac.Role
	Rating          int
	IsActive        bool
	IsSuspended     bool
	AvatarURL       *string
	AvatarObjectKey *string
	Bio             *string
	Country         *string
	School          *string
	Company         *string
	GithubURL       *string
	WebsiteURL      *string
	LinkedinURL     *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
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

func (u *User) Suspend() {
	if u.IsSuspended {
		return
	}
	u.IsSuspended = true
	u.UpdatedAt = time.Now()
}

func (u *User) Unsuspend() {
	if !u.IsSuspended {
		return
	}
	u.IsSuspended = false
	u.UpdatedAt = time.Now()
}

func (u *User) UpdatePassword(newPassword valueobject.Password) {
	u.Password = newPassword.Hash()
	u.UpdatedAt = time.Now()
}

func (u *User) UploadAvatar(avatarURL string, avatarObjectKey string) {
	u.AvatarURL = &avatarURL
	u.AvatarObjectKey = &avatarObjectKey
	u.UpdatedAt = time.Now()
}

func (u *User) UpdateProfile(
	fullName *string,
	bio *string,
	country *string,
	school *string,
	company *string,
	githubURL *string,
	websiteURL *string,
	linkedinURL *string,
) {
	if fullName != nil {
		u.FullName = *fullName
	}
	if bio != nil {
		u.Bio = bio
	}
	if country != nil {
		u.Country = country
	}
	if school != nil {
		u.School = school
	}
	if company != nil {
		u.Company = company
	}
	if githubURL != nil {
		u.GithubURL = githubURL
	}
	if websiteURL != nil {
		u.WebsiteURL = websiteURL
	}
	if linkedinURL != nil {
		u.LinkedinURL = linkedinURL
	}

	u.UpdatedAt = time.Now()
}

func (u *User) AssignRole(role rbac.Role) {
	u.Role = role
	u.UpdatedAt = time.Now()
}
