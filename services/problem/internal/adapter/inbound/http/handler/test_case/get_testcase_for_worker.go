package testcase

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type GetTestCaseForWorkerHandler struct {
	uc inbound.GetTestCaseForWorkerUseCase
}

func NewGetTestCaseForWorkerHandler(uc inbound.GetTestCaseForWorkerUseCase) *GetTestCaseForWorkerHandler {
	return &GetTestCaseForWorkerHandler{uc: uc}
}

func (h *GetTestCaseForWorkerHandler) Handle(c *gin.Context) {
	response.HandleWithParams(c, h.uc.Execute, response.CodeSuccess)
}
