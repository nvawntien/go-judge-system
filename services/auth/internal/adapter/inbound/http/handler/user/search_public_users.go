package user

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type SearchPublicUsersHandler struct {
	uc inbound.SearchPublicUsersUseCase
}

func NewSearchPublicUsersHandler(uc inbound.SearchPublicUsersUseCase) *SearchPublicUsersHandler {
	return &SearchPublicUsersHandler{uc: uc}
}

func (h *SearchPublicUsersHandler) Handle(c *gin.Context) {
	response.HandleWithQuery(c, h.uc.Execute, response.CodeSuccess)
}
