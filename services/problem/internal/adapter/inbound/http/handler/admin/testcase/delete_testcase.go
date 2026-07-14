package testcase

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type DeleteTestCaseHandler struct {
	uc inbound.DeleteTestCaseUseCase
}

func NewDeleteTestCaseHandler(uc inbound.DeleteTestCaseUseCase) *DeleteTestCaseHandler {
	return &DeleteTestCaseHandler{uc: uc}
}

func (h *DeleteTestCaseHandler) Handle(c *gin.Context) {
	response.HandleVoidWithParamsAndClaims(c, h.uc.Execute, response.CodeSuccess, "testcase deleted successfully")
}
