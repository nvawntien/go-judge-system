package user

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"

	"github.com/gin-gonic/gin"
)

type IssueSubmissionStreamTicketHandler struct {
	uc inbound.IssueSubmissionStreamTicketUseCase
}

func NewIssueSubmissionStreamTicketHandler(
	uc inbound.IssueSubmissionStreamTicketUseCase,
) *IssueSubmissionStreamTicketHandler {
	return &IssueSubmissionStreamTicketHandler{uc: uc}
}

func (h *IssueSubmissionStreamTicketHandler) Handle(c *gin.Context) {
	response.HandleWithParamsAndClaims(
		c,
		h.uc.Execute,
		response.CodeSuccess,
	)
}
