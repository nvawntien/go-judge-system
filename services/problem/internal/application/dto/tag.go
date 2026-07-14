package dto

type TagIDRequest struct {
	ID uint `uri:"tag_id" binding:"required,min=1"`
}

type TagResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
}

type AdminTagResponse struct {
	TagResponse
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ListTagsResponse struct {
	Items []TagResponse `json:"items"`
}

type AdminListTagsResponse struct {
	Items []AdminTagResponse `json:"items"`
}

type CreateTagRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=64"`
	Slug        string `json:"slug,omitempty" binding:"omitempty,min=1,max=64"`
	Description string `json:"description,omitempty" binding:"omitempty,max=255"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

type UpdateTagRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=64"`
	Slug        *string `json:"slug,omitempty" binding:"omitempty,min=1,max=64"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=255"`
	IsActive    *bool   `json:"is_active,omitempty"`
}
