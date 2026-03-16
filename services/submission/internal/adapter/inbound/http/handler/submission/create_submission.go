package submission

import (
	"go-judge-system/pkg/response"
	"go-judge-system/services/submission/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type CreateSubmissionHandler struct {
	uc inbound.CreateSubmissionUseCase
}

func NewCreateSubmissionHandler(uc inbound.CreateSubmissionUseCase) *CreateSubmissionHandler {
	return &CreateSubmissionHandler{uc: uc}
}

func (h *CreateSubmissionHandler) Handle(c *gin.Context) {
	response.HandleWithClaims(c, h.uc.Execute, response.CodeCreated)
}
