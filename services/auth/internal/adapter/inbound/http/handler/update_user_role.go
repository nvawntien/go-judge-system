package handler

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type UpdateUserRoleHandler struct {
	uc inbound.UpdateUserRoleUseCase
}

func NewUpdateUserRoleHandler(uc inbound.UpdateUserRoleUseCase) *UpdateUserRoleHandler {
	return &UpdateUserRoleHandler{uc: uc}
}

func (h *UpdateUserRoleHandler) Handle(c *gin.Context) {
	response.HandleVoidWithParamsAndBody(c, h.uc.Execute, response.CodeUpdated, "user role updated successfully")
}