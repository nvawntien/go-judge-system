package testcase

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type UpdateTestCaseHandler struct {
	uc inbound.UpdateTestCaseUseCase
}

func NewUpdateTestCaseHandler(uc inbound.UpdateTestCaseUseCase) *UpdateTestCaseHandler {
	return &UpdateTestCaseHandler{uc: uc}
}

func (h *UpdateTestCaseHandler) Handle(c *gin.Context) {
	response.HandleVoidWithParamsAndBody(c, h.uc.Execute, response.CodeUpdated, "test case updated successfully")
}
