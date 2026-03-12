package problem

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type HideProblemHandler struct {
	uc inbound.HideProblemUseCase
}

func NewHideProblemHandler(uc inbound.HideProblemUseCase) *HideProblemHandler {
	return &HideProblemHandler{uc: uc}
}

func (h *HideProblemHandler) Handle(c *gin.Context) {
	response.HandleVoidWithParamsAndClaims(c, h.uc.Execute, response.CodeUpdated, "problem hidden successfully")
}
