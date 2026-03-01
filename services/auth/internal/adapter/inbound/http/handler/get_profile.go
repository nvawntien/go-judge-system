package handler

import (
	"errors"
	"go-judge-system/services/auth/internal/adapter/inbound/http/response"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GetProfileHandler struct {
	uc inbound.GetProfileUseCase
}

func NewGetProfileHandler(uc inbound.GetProfileUseCase) *GetProfileHandler {
	return &GetProfileHandler{uc: uc}
}

func (h *GetProfileHandler) HandleMe(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		response.Error(c, http.StatusUnauthorized, "Unauthorized")
		return
	}

	profile, err := h.uc.Execute(c, username)
	if err != nil {

		response.Error(c, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.Success(c, http.StatusOK, profile)
}

func (h *GetProfileHandler) HandlePublic(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		response.Error(c, http.StatusBadRequest, "Bad Request")
		return
	}

	profile, err := h.uc.Execute(c, username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.Error(c, http.StatusNotFound, "User not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response.Success(c, http.StatusOK, profile)
}
