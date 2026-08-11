package admin

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type RejudgeSubmissionHandler struct {
	uc inbound.RejudgeAdminSubmissionUseCase
}

func NewRejudgeSubmissionHandler(uc inbound.RejudgeAdminSubmissionUseCase) *RejudgeSubmissionHandler {
	return &RejudgeSubmissionHandler{uc: uc}
}

func (h *RejudgeSubmissionHandler) Handle(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	response.HandleWithParamsAndClaims(
		c,
		h.uc.Execute,
		response.CodeSuccess,
	)
}
