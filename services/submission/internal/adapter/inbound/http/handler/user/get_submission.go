package user

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"

	"github.com/gin-gonic/gin"
)

type GetSubmissionHandler struct {
	uc inbound.GetSubmissionUseCase
}

func NewGetSubmissionHandler(uc inbound.GetSubmissionUseCase) *GetSubmissionHandler {
	return &GetSubmissionHandler{uc: uc}
}

func (h *GetSubmissionHandler) Handle(c *gin.Context) {
	response.HandleWithParamsAndClaims(
		c,
		h.uc.Execute,
		response.CodeSuccess,
	)
}
