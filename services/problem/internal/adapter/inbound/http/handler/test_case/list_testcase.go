package testcase

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type ListTestCasesHandler struct {
	uc inbound.ListTestCasesUseCase
}

func NewListTestCasesHandler(uc inbound.ListTestCasesUseCase) *ListTestCasesHandler {
	return &ListTestCasesHandler{uc: uc}
}

func (h *ListTestCasesHandler) Handle(c *gin.Context) {
	response.HandleWithParamsAndClaims(c, h.uc.Execute, response.CodeSuccess)
}
