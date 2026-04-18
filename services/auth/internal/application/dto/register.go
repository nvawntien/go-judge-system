package dto

type RegisterRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}