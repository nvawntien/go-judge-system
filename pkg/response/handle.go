package response

import (
	"context"
	"errors"
	"go-judge-system/pkg/auth"

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


// HandleVoidWithClaims: Used for APIs that receive JSON body data, but do not return data and require authentication.
func HandleVoidWithClaims[Req any](c *gin.Context, fn func(context.Context, auth.Claims, Req) error, successCode int, successMsg string) {
	claims, ok := auth.GetClaims(c)
	if !ok {
		Error(c, CodeUnauthorized, "unauthorized")
		return
	}

	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeBadRequest, "invalid request payload")
		return
	}

	if err := fn(c.Request.Context(), claims, req); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, successCode, successMsg, nil)
}

// HandleVoidWithParams: Used for APIs that receive URI params, but do not return data.
func HandleVoidWithParams[Req any](c *gin.Context, fn func(context.Context, Req) error, successCode int, successMsg string) {
	var req Req
	if err := c.ShouldBindUri(&req); err != nil {
		Error(c, CodeParamInvalid, "invalid uri params")
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

// HandleWithClaims: Used for APIs that receive JSON body data and return data and require authentication.
func HandleWithClaims[Req any, Res any](c *gin.Context, fn func(context.Context, auth.Claims, Req) (Res, error), successCode int) {
	claims, ok := auth.GetClaims(c)
	if !ok {
		Error(c, CodeUnauthorized, "unauthorized")
		return
	}

	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeBadRequest, "invalid request payload")
		return
	}

	res, err := fn(c.Request.Context(), claims, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, successCode, res)
}

// HandleWithClaimsNoBody: Used for APIs that not JSON body data and return data and require authentication.
func HandleWithClaimsNoBody[Res any](c *gin.Context, fn func(context.Context, auth.Claims) (Res, error), successCode int) {
	claims, ok := auth.GetClaims(c)
	if !ok {
		Error(c, CodeUnauthorized, "unauthorized")
		return
	}

	res, err := fn(c.Request.Context(), claims)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, successCode, res)
}

// HandleWithParams: Used for APIs that receive URI params and return data.
func HandleWithParams[Req any, Res any](c *gin.Context, fn func(context.Context, Req) (Res, error), successCode int) {
	var req Req
	if err := c.ShouldBindUri(&req); err != nil {
		Error(c, CodeParamInvalid, "invalid uri params")
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
