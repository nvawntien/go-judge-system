package problem

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type PublishProblemHandler struct{
	uc inbound.PublishProblemUseCase
}

func NewPublishProblemHandler(uc inbound.PublishProblemUseCase) *PublishProblemHandler {
	return &PublishProblemHandler{uc: uc}
}

func (h *PublishProblemHandler) Handle(c *gin.Context) {
	response.HandleVoidWithParamsAndClaims(c, h.uc.Execute, response.CodeUpdated, "problem published successfully")
}
