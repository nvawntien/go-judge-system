package user

import (
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/domain/entity"
)

func toGetMeResponse(user *entity.User) *dto.GetMeResponse {
	return &dto.GetMeResponse{
		ID:          user.ID,
		FullName:    user.FullName,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		Rating:      user.Rating,
		IsActive:    user.IsActive,
		AvatarURL:   user.AvatarURL,
		Bio:         user.Bio,
		Country:     user.Country,
		School:      user.School,
		Company:     user.Company,
		GithubURL:   user.GithubURL,
		WebsiteURL:  user.WebsiteURL,
		LinkedinURL: user.LinkedinURL,
		CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toGetProfileResponse(user *entity.User) dto.GetProfileResponse {
	return dto.GetProfileResponse{
		FullName:    user.FullName,
		Username:    user.Username,
		Rating:      user.Rating,
		AvatarURL:   user.AvatarURL,
		Bio:         user.Bio,
		Country:     user.Country,
		School:      user.School,
		Company:     user.Company,
		GithubURL:   user.GithubURL,
		WebsiteURL:  user.WebsiteURL,
		LinkedinURL: user.LinkedinURL,
		CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
