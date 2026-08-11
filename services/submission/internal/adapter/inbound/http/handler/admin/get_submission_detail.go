package admin

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type GetSubmissionDetailHandler struct {
	uc inbound.GetAdminSubmissionDetailUseCase
}

func NewGetSubmissionDetailHandler(uc inbound.GetAdminSubmissionDetailUseCase) *GetSubmissionDetailHandler {
	return &GetSubmissionDetailHandler{uc: uc}
}

func (h *GetSubmissionDetailHandler) Handle(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	response.HandleWithParamsAndClaims(
		c,
		h.uc.Execute,
		response.CodeSuccess,
	)
}
