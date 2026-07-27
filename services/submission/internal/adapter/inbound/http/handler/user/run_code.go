package user

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"

	"github.com/gin-gonic/gin"
)

type RunCodeHandler struct {
	uc inbound.RunCodeUseCase
}

func NewRunCodeHandler(uc inbound.RunCodeUseCase) *RunCodeHandler {
	return &RunCodeHandler{uc: uc}
}

func (h *RunCodeHandler) Handle(c *gin.Context) {
	response.HandleWithClaims(c, h.uc.Execute, response.CodeSuccess)
}
