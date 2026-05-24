package handler

import "go-judge-system/services/auth/internal/adapter/inbound/http/handler/admin"

type AdminHandler struct {
	AssignRole *admin.AssignRoleHandler
}

func NewAdminHandler(assignRole *admin.AssignRoleHandler) *AdminHandler {
	return &AdminHandler{
		AssignRole: assignRole,
	}
}