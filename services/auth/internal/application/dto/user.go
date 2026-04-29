package dto

type GetMeRequest struct {
}

type GetMeResponse struct {
	ID        string `json:"id"`
	FullName  string `json:"full_name"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Rating    int    `json:"rating"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
