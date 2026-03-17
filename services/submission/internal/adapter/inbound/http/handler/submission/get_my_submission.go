package submission

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type GetSubmissionHandler struct {
	uc inbound.GetSubmissionUseCase
}

func NewGetSubmissionHandler(uc inbound.GetSubmissionUseCase) *GetSubmissionHandler {
	return &GetSubmissionHandler{uc: uc}
}

func (h *GetSubmissionHandler) HandleMy(c *gin.Context) {
	response.HandleWithParamsAndClaims(c, h.uc.ExecuteMy, response.CodeSuccess)
}

func (h *GetSubmissionHandler) HandleAdmin(c *gin.Context) {
	response.HandleWithParamsAndClaims(c, h.uc.ExecuteAdmin, response.CodeSuccess)
}
