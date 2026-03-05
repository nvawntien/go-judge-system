package response

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
)

// HandleVoid: Used for APIs that receive JSON body data, but do not return data.
func HandleVoid[Req any](c *gin.Context, fn func(context.Context, Req) error, successCode int, successMsg string) {
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeBadRequest, "invalid request payload")
		return
	}

	if err := fn(c.Request.Context(), req); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, successCode, successMsg, nil)
}

// Handle: Used for APIs that receive JSON body data and return data.
func Handle[Req any, Res any](c *gin.Context, fn func(context.Context, Req) (Res, error), successCode int) {
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeBadRequest, "invalid request payload")
		return
	}

	res, err := fn(c.Request.Context(), req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, successCode, res)
}

// HandleWithMessage: Used for APIs that receive JSON body data and return data with a custom success message.
func HandleWithMessage[Req any, Res any](c *gin.Context, fn func(context.Context, Req) (Res, error), successCode int, successMsg string) {
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeBadRequest, "invalid request payload")
		return
	}

	res, err := fn(c.Request.Context(), req)
	if err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, successCode, successMsg, res)
}

// HandleError: Handles errors and returns appropriate HTTP responses.
func HandleError(c *gin.Context, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		Error(c, appErr.Code, appErr.Message)
		return
	}

	Error(c, CodeInternalServer, "internal server error")
}
