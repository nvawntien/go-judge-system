package user

import (
	"go-judge-system/pkg/response"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"

	"github.com/gin-gonic/gin"
)

type CreateSubmissionHandler struct {
	uc inbound.CreateSubmissionUseCase
}

func NewCreateSubmissionHandler(uc inbound.CreateSubmissionUseCase) *CreateSubmissionHandler {
	return &CreateSubmissionHandler{uc: uc}
}

func (h *CreateSubmissionHandler) Handle(c *gin.Context) {
	response.HandleWithClaims(
		c,
		h.uc.Execute,
		response.CodeCreated,
	)
}
