package testcase

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type UploadTestCaseHandler struct {
	uc inbound.UploadTestCaseUseCase
}

func NewUploadTestCaseHandler(uc inbound.UploadTestCaseUseCase) *UploadTestCaseHandler {
	return &UploadTestCaseHandler{uc: uc}
}

func (h *UploadTestCaseHandler) Handle(c *gin.Context) {
	response.HandleWithParamsAndForm(c, h.uc.Execute, response.CodeCreated)
}
