package problem

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type ListProblemsHandler struct{
	uc inbound.ListProblemsUseCase
}

func NewListProblemsHandler(uc inbound.ListProblemsUseCase) *ListProblemsHandler {
	return &ListProblemsHandler{uc: uc}
}

func (h *ListProblemsHandler) Handle(c *gin.Context) {
	response.HandleWithQuery(c, h.uc.Execute, response.CodeSuccess)
}

func (h *ListProblemsHandler) HandleMy(c *gin.Context) {
	response.HandleWithQueryAndClaims(c, h.uc.ExecuteMy, response.CodeSuccess)
}

func (h *ListProblemsHandler) HandleAdmin(c *gin.Context) {
	response.HandleWithQueryAndClaims(c, h.uc.ExecuteAdmin, response.CodeSuccess)
}
