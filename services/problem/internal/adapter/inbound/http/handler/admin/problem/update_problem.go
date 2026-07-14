package problem

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type UpdateProblemHandler struct {
	uc inbound.UpdateProblemUseCase
}

func NewUpdateProblemHandler(uc inbound.UpdateProblemUseCase) *UpdateProblemHandler {
	return &UpdateProblemHandler{uc: uc}
}

func (h *UpdateProblemHandler) Handle(c *gin.Context) {
	response.HandleWithClaimsParamsAndBody(c, h.uc.Execute, response.CodeSuccess)
}
