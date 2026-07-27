package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/admin"

type AdminHandler struct {
	ListSubmissions *admin.ListSubmissionsHandler
}

func NewAdminHandler(
	listSubmissions *admin.ListSubmissionsHandler,
) *AdminHandler {
	return &AdminHandler{ListSubmissions: listSubmissions}
}
