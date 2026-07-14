package entity

import "time"

type Tag struct {
	ID          uint
	Name        string
	Slug        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewTag(name, slug, description string, isActive bool) *Tag {
	return &Tag{
		Name:        name,
		Slug:        slug,
		Description: description,
		IsActive:    isActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
