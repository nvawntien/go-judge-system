package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/admin"

type AdminHandler struct {
	ListSubmissions     *admin.ListSubmissionsHandler
	GetSubmissionDetail *admin.GetSubmissionDetailHandler
}

func NewAdminHandler(
	listSubmissions *admin.ListSubmissionsHandler,
	getSubmissionDetail *admin.GetSubmissionDetailHandler,
) *AdminHandler {
	return &AdminHandler{
		ListSubmissions:     listSubmissions,
		GetSubmissionDetail: getSubmissionDetail,
	}
}
