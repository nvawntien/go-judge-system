package problem

import (
	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/dto"
	"go-judge-system/services/problem/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

type UpdateProblemHandler struct {
	uc inbound.UpdateProblemUseCase
}

func NewUpdateProblemHandler(uc inbound.UpdateProblemUseCase) *UpdateProblemHandler {
	return &UpdateProblemHandler{uc: uc}
}

func (h *UpdateProblemHandler) Handle(c *gin.Context) {
	claims, ok := auth.GetClaims(c)
	if !ok {
		response.HandleError(c, response.NewAppError(response.CodeUnauthorized, "unauthorized", nil))
		return
	}

	var params dto.ProblemIDRequest
	if err := c.ShouldBindUri(&params); err != nil {
		response.HandleError(c, response.NewAppError(response.CodeParamInvalid, "invalid uri params", err))
		return
	}

	var body dto.UpdateProblemRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.HandleError(c, response.NewAppError(response.CodeBadRequest, "invalid request payload", err))
		return
	}

	if err := h.uc.Execute(c.Request.Context(), claims, params, body); err != nil {
		response.HandleError(c, err)
		return
	}

	response.SuccessWithMessage(c, response.CodeUpdated, "problem updated successfully", nil)
}
