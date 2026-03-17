package submission

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type RejudgeSubmissionHandler struct {
	uc inbound.RejudgeSubmissionUseCase
}

func NewRejudgeSubmissionHandler(uc inbound.RejudgeSubmissionUseCase) *RejudgeSubmissionHandler {
	return &RejudgeSubmissionHandler{uc: uc}
}

func (h *RejudgeSubmissionHandler) Handle(c *gin.Context) {
	response.HandleVoidWithParamsAndClaims(c, h.uc.Execute, response.CodeUpdated, "Submission rejudge queued")
}
