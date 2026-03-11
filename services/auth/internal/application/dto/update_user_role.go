package dto

type UpdateUserRoleRequest struct {
	Username string  `uri:"username" binding:"required"`
	Role     *string `json:"role" binding:"omitempty,oneof=admin user"`
}
