package tag

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/user"

	"github.com/gin-gonic/gin"
)

type ListTagsHandler struct {
	uc inbound.ListTagsUseCase
}

func NewListTagsHandler(uc inbound.ListTagsUseCase) *ListTagsHandler {
	return &ListTagsHandler{uc: uc}
}

func (h *ListTagsHandler) Handle(c *gin.Context) {
	response.HandleNoBody(c, h.uc.Execute, response.CodeSuccess)
}
