package user

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type UploadAvatarHandler struct {
	uc inbound.UploadAvatarUseCase
}

func NewUploadAvatarHandler(uc inbound.UploadAvatarUseCase) *UploadAvatarHandler {
	return &UploadAvatarHandler{uc: uc}
}

func (h *UploadAvatarHandler) Handle(c *gin.Context) {
	response.HandleWithFormAndClaims(
		c,
		h.uc.Execute,
		response.CodeSuccess,
	)
}
