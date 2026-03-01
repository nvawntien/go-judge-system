package dto

import "time"

type ProfileResponse struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Rating    int       `json:"rating"`
	CreatedAt time.Time `json:"created_at"`
}