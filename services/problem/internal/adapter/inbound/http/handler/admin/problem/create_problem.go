package problem

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type CreateProblemHandler struct {
	uc inbound.CreateProblemUseCase
}

func NewCreateProblemHandler(uc inbound.CreateProblemUseCase) *CreateProblemHandler {
	return &CreateProblemHandler{uc: uc}
}

func (h *CreateProblemHandler) Handle(c *gin.Context) {
	response.HandleWithClaims(c, h.uc.Execute, response.CodeCreated)
}
