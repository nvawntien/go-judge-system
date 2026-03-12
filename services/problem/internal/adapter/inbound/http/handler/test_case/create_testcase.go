package testcase

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type CreateTestCaseHandler struct{
	uc inbound.CreateTestCaseUseCase
}

func NewCreateTestCaseHandler(uc inbound.CreateTestCaseUseCase) *CreateTestCaseHandler {
	return &CreateTestCaseHandler{uc: uc}
}

func (h *CreateTestCaseHandler) Handle(c *gin.Context) {
	response.HandleWithParamsAndBody(c, h.uc.Execute, response.CodeCreated)
}
