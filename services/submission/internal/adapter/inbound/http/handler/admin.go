package handler

import "go-judge-system/services/submission/internal/adapter/inbound/http/handler/admin"

type AdminHandler struct {
	ListSubmissions     *admin.ListSubmissionsHandler
	GetSubmissionDetail *admin.GetSubmissionDetailHandler
	RejudgeSubmission   *admin.RejudgeSubmissionHandler
}

func NewAdminHandler(
	listSubmissions *admin.ListSubmissionsHandler,
	getSubmissionDetail *admin.GetSubmissionDetailHandler,
	rejudgeSubmissions ...*admin.RejudgeSubmissionHandler,
) *AdminHandler {
	var rejudgeSubmission *admin.RejudgeSubmissionHandler
	if len(rejudgeSubmissions) > 0 {
		rejudgeSubmission = rejudgeSubmissions[0]
	}
	return &AdminHandler{
		ListSubmissions:     listSubmissions,
		GetSubmissionDetail: getSubmissionDetail,
		RejudgeSubmission:   rejudgeSubmission,
	}
}
