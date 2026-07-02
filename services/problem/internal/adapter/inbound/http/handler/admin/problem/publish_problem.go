package problem

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type PublishProblemHandler struct {
	uc inbound.PublishProblemUseCase
}

func NewPublishProblemHandler(uc inbound.PublishProblemUseCase) *PublishProblemHandler {
	return &PublishProblemHandler{uc: uc}
}

func (h *PublishProblemHandler) Handle(c *gin.Context) {
	response.HandleWithParams(c, h.uc.Execute, response.CodeSuccess)
}
