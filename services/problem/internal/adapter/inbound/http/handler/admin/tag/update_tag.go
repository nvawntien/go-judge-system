package tag

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type UpdateTagHandler struct {
	uc inbound.UpdateTagUseCase
}

func NewUpdateTagHandler(uc inbound.UpdateTagUseCase) *UpdateTagHandler {
	return &UpdateTagHandler{uc: uc}
}

func (h *UpdateTagHandler) Handle(c *gin.Context) {
	response.HandleWithClaimsParamsAndBody(c, h.uc.Execute, response.CodeSuccess)
}
