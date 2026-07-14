package testcase

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type GetTestCaseHandler struct {
	uc inbound.GetTestCaseUseCase
}

func NewGetTestCaseHandler(uc inbound.GetTestCaseUseCase) *GetTestCaseHandler {
	return &GetTestCaseHandler{uc: uc}
}

func (h *GetTestCaseHandler) Handle(c *gin.Context) {
	response.HandleWithParamsAndClaims(c, h.uc.Execute, response.CodeSuccess)
}
