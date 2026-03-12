package problem

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type GetProblemHandler struct{
	uc inbound.GetProblemUseCase
}

func NewGetProblemHandler(uc inbound.GetProblemUseCase) *GetProblemHandler {
	return &GetProblemHandler{uc: uc}
}

func (h *GetProblemHandler) Handle(c *gin.Context) {
	response.HandleWithParams(c, h.uc.Execute, response.CodeSuccess)
}

func (h *GetProblemHandler) HandleAdmin(c *gin.Context) {
	response.HandleWithParamsAndClaims(c, h.uc.ExecuteAdmin, response.CodeSuccess)
}

