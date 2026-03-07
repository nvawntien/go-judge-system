package dto

import "time"

type ProfileRequest struct {
	Username string `uri:"username" binding:"required"`
}

type ProfileResponse struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Rating    int       `json:"rating"`
	CreatedAt time.Time `json:"created_at"`
}