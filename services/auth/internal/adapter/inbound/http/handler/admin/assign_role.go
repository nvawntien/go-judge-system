package admin

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type AssignRoleHandler struct {
	uc inbound.AssignRoleUseCase
}

func NewAssignRoleHandler(uc inbound.AssignRoleUseCase) *AssignRoleHandler {
	return &AssignRoleHandler{uc: uc}
}

func (h *AssignRoleHandler) Handle(c *gin.Context) {
	response.HandleVoidWithParamsAndBody(
		c,
		h.uc.Execute,
		response.CodeSuccess,
		"Assign role for user successfully",
	)
}
