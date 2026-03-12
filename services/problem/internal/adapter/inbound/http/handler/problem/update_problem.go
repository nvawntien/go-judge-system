package problem

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type UpdateProblemHandler struct{
	uc inbound.UpdateProblemUseCase
}

func NewUpdateProblemHandler(uc inbound.UpdateProblemUseCase) *UpdateProblemHandler {
	return &UpdateProblemHandler{uc: uc}
}

func (h *UpdateProblemHandler) Handle(c *gin.Context) {
	response.HandleVoidWithParamsAndBody(c, h.uc.Execute, response.CodeUpdated, "problem updated successfully")
}
