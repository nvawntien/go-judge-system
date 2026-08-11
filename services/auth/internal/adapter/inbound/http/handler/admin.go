package handler

import "go-judge-system/services/auth/internal/adapter/inbound/http/handler/admin"

type AdminHandler struct {
	AssignRole *admin.AssignRoleHandler
	Users      *admin.AdminUsersHandler
}

func NewAdminHandler(assignRole *admin.AssignRoleHandler, users *admin.AdminUsersHandler) *AdminHandler {
	return &AdminHandler{
		AssignRole: assignRole,
		Users:      users,
	}
}
