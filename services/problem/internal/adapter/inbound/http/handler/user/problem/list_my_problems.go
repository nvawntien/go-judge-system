package problem

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/user"

	"github.com/gin-gonic/gin"
)

type ListMyProblemsHandler struct {
	uc inbound.ListMyProblemsUseCase
}

func NewListMyProblemsHandler(uc inbound.ListMyProblemsUseCase) *ListMyProblemsHandler {
	return &ListMyProblemsHandler{uc: uc}
}

func (h *ListMyProblemsHandler) Handle(c *gin.Context) {
	response.HandleWithQueryAndClaims(c, h.uc.Execute, response.CodeSuccess)
}
