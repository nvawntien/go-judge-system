package dto

type UserRoleRequest struct {
	Username string `uri:"username" binding:"required"`
}

type UpdateUserRoleRequest struct {
	Role *string `json:"role" binding:"omitempty,oneof=admin user"`
}
