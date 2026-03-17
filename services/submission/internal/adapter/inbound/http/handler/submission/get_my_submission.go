package submission

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type GetMySubmissionHandler struct {
	uc inbound.GetSubmissionUseCase
}

func NewGetMySubmissionHandler(uc inbound.GetSubmissionUseCase) *GetMySubmissionHandler {
	return &GetMySubmissionHandler{uc: uc}
}

func (h *GetMySubmissionHandler) Handle(c *gin.Context) {
	response.HandleWithParamsAndClaims(c, h.uc.ExecuteMy, response.CodeSuccess)
}
