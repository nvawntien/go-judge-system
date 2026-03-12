package problem

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type DeleteProblemHandler struct {
	uc inbound.DeleteProblemUseCase
}

func NewDeleteProblemHandler(uc inbound.DeleteProblemUseCase) *DeleteProblemHandler {
	return &DeleteProblemHandler{uc: uc}
}

func (h *DeleteProblemHandler) Handle(c *gin.Context) {
	response.HandleVoidWithParamsAndClaims(c, h.uc.Execute, response.CodeDeleted, "problem deleted successfully")
}
