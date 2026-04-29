package user

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type GetMeHandler struct {
	uc inbound.GetMeUseCase
}

func NewGetMeHandler(uc inbound.GetMeUseCase) *GetMeHandler {
	return &GetMeHandler{uc: uc}
}

func (h *GetMeHandler) Handle(c *gin.Context) {
	response.HandleWithClaimsNoBody(
		c,
		h.uc.Execute,
		response.CodeSuccess,
	)
}
