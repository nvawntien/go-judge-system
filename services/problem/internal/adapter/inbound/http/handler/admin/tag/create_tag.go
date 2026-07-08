package tag

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"

	"github.com/gin-gonic/gin"
)

type CreateTagHandler struct {
	uc inbound.CreateTagUseCase
}

func NewCreateTagHandler(uc inbound.CreateTagUseCase) *CreateTagHandler {
	return &CreateTagHandler{uc: uc}
}

func (h *CreateTagHandler) Handle(c *gin.Context) {
	response.HandleWithClaims(c, h.uc.Execute, response.CodeCreated)
}
