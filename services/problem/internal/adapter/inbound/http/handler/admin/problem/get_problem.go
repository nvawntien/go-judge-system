package problem

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type GetProblemHandler struct {
	uc inbound.GetProblemUseCase
}

func NewGetProblemHandler(uc inbound.GetProblemUseCase) *GetProblemHandler {
	return &GetProblemHandler{uc: uc}
}

func (h *GetProblemHandler) Handle(c *gin.Context) {
	response.HandleWithParamsAndClaims(c, h.uc.Execute, response.CodeSuccess)
}
