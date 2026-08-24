package problem

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type HiddenProblemHandler struct {
	uc inbound.HiddenProblemUseCase
}

func NewHiddenProblemHandler(uc inbound.HiddenProblemUseCase) *HiddenProblemHandler {
	return &HiddenProblemHandler{uc: uc}
}

func (h *HiddenProblemHandler) Handle(c *gin.Context) {
	response.HandleWithParamsAndClaims(c, h.uc.Execute, response.CodeSuccess)
}
