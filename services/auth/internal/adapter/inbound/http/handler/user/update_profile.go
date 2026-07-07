package user

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type UpdateProfileHandler struct {
	uc inbound.UpdateProfileUseCase
}

func NewUpdateProfileHandler(uc inbound.UpdateProfileUseCase) *UpdateProfileHandler {
	return &UpdateProfileHandler{uc: uc}
}

func (h *UpdateProfileHandler) Handle(c *gin.Context) {
	response.HandleWithClaims(
		c,
		h.uc.Execute,
		response.CodeSuccess,
	)
}
