package testcase

import (
	"go-judge-system/pkg/response"

	"github.com/gin-gonic/gin"
)

import inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"

type UploadTestCaseHandler struct {
	uc inbound.UploadTestCaseUseCase
}

func NewUploadTestCaseHandler(uc inbound.UploadTestCaseUseCase) *UploadTestCaseHandler {
	return &UploadTestCaseHandler{uc: uc}
}

func (h *UploadTestCaseHandler) Handle(c *gin.Context) {
	response.HandleWithClaimsParamsAndForm(c, h.uc.Execute, response.CodeCreated)
}
