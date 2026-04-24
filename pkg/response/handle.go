package response

import (
	"context"
	"errors"
	"go-judge-system/pkg/auth"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var errorLogger *zap.Logger

// SetErrorLogger configures the global logger used by HandleError
// to automatically log root cause errors. Call once at application startup.
func SetErrorLogger(l *zap.Logger) {
	errorLogger = l
}

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

// HandleVoidWithParamsAndClaims: URI params + claims → void. Used for DELETE, Publish, Hide.
func HandleVoidWithParamsAndClaims[Req any](c *gin.Context, fn func(context.Context, auth.Claims, Req) error, successCode int, successMsg string) {
	claims, ok := auth.GetClaims(c)
	if !ok {
		Error(c, CodeUnauthorized, "unauthorized")
		return
	}

	var req Req
	if err := c.ShouldBindUri(&req); err != nil {
		Error(c, CodeParamInvalid, "invalid uri params")
		return
	}

	if err := fn(c.Request.Context(), claims, req); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, successCode, successMsg, nil)
}

// HandleWithParamsAndClaims: URI params + claims → data. Used for admin GET by ID.
func HandleWithParamsAndClaims[Req any, Res any](c *gin.Context, fn func(context.Context, auth.Claims, Req) (Res, error), successCode int) {
	claims, ok := auth.GetClaims(c)
	if !ok {
		Error(c, CodeUnauthorized, "unauthorized")
		return
	}

	var req Req
	if err := c.ShouldBindUri(&req); err != nil {
		Error(c, CodeParamInvalid, "invalid uri params")
		return
	}

	res, err := fn(c.Request.Context(), claims, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, successCode, res)
}

// HandleWithQuery: Query params → data. Used for public list endpoints.
func HandleWithQuery[Req any, Res any](c *gin.Context, fn func(context.Context, Req) (Res, error), successCode int) {
	var req Req
	if err := c.ShouldBindQuery(&req); err != nil {
		Error(c, CodeBadRequest, "invalid query parameters")
		return
	}

	res, err := fn(c.Request.Context(), req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, successCode, res)
}

// HandleWithParamsAndQuery: URI params + query params (bound separately) -> data.
func HandleWithParamsAndQuery[P any, Q any, Res any](c *gin.Context, fn func(context.Context, P, Q) (Res, error), successCode int) {
	var params P
	if err := c.ShouldBindUri(&params); err != nil {
		Error(c, CodeParamInvalid, "invalid uri params")
		return
	}

	var query Q
	if err := c.ShouldBindQuery(&query); err != nil {
		Error(c, CodeBadRequest, "invalid query parameters")
		return
	}

	res, err := fn(c.Request.Context(), params, query)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, successCode, res)
}

// HandleWithQueryAndClaims: Query params + claims → data. Used for admin list, my list.
func HandleWithQueryAndClaims[Req any, Res any](c *gin.Context, fn func(context.Context, auth.Claims, Req) (Res, error), successCode int) {
	claims, ok := auth.GetClaims(c)
	if !ok {
		Error(c, CodeUnauthorized, "unauthorized")
		return
	}

	var req Req
	if err := c.ShouldBindQuery(&req); err != nil {
		Error(c, CodeBadRequest, "invalid query parameters")
		return
	}

	res, err := fn(c.Request.Context(), claims, req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, successCode, res)
}

// HandleVoidWithParamsAndBody: URI params + JSON body + claims → void. Used for Update (ID + body).
func HandleVoidWithParamsAndBody[P any, B any](c *gin.Context, fn func(context.Context, auth.Claims, P, B) error, successCode int, successMsg string) {
	claims, ok := auth.GetClaims(c)
	if !ok {
		Error(c, CodeUnauthorized, "unauthorized")
		return
	}

	var params P

	if err := c.ShouldBindUri(&params); err != nil {
		Error(c, CodeParamInvalid, "invalid uri params")
		return
	}

	var body B

	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, CodeBadRequest, "invalid request payload")
		return
	}

	if err := fn(c.Request.Context(), claims, params, body); err != nil {
		HandleError(c, err)
		return
	}

	SuccessWithMessage(c, successCode, successMsg, nil)
}

// HandleWithParamsAndBody: URI params + JSON body + claims → data. Used for Create TestCase (problem ID + body).
func HandleWithParamsAndBody[P any, B any, Res any](c *gin.Context, fn func(context.Context, auth.Claims, P, B) (Res, error), successCode int) {
	claims, ok := auth.GetClaims(c)
	if !ok {
		Error(c, CodeUnauthorized, "unauthorized")
		return
	}

	var params P
	if err := c.ShouldBindUri(&params); err != nil {
		Error(c, CodeParamInvalid, "invalid uri params")
		return
	}

	var body B
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, CodeBadRequest, "invalid request payload")
		return
	}

	res, err := fn(c.Request.Context(), claims, params, body)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, successCode, res)
}

// HandleWithParamsAndForm: URI params + multipart form/form-data + claims → data.
// Identical to HandleWithParamsAndBody but uses ShouldBind (auto-detects content type)
// instead of ShouldBindJSON. Used for file upload endpoints (e.g. Upload TestCase ZIP).
func HandleWithParamsAndForm[P any, F any, Res any](c *gin.Context, fn func(context.Context, auth.Claims, P, F) (Res, error), successCode int) {
	claims, ok := auth.GetClaims(c)
	if !ok {
		Error(c, CodeUnauthorized, "unauthorized")
		return
	}

	var params P
	if err := c.ShouldBindUri(&params); err != nil {
		Error(c, CodeParamInvalid, "invalid uri params")
		return
	}

	var form F
	if err := c.ShouldBind(&form); err != nil {
		Error(c, CodeBadRequest, "invalid form data")
		return
	}

	res, err := fn(c.Request.Context(), claims, params, form)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, successCode, res)
}

// HandleError: Handles errors and returns appropriate HTTP responses.
// If an AppError carries a root cause (via Wrap), it is automatically logged
// with request context and origin stack trace.
func HandleError(c *gin.Context, err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		// Log internal root cause if present (server errors wrapped via .Wrap())
		if appErr.Err != nil && errorLogger != nil {
			errorLogger.Error("request error",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Int("code", appErr.Code),
				zap.String("message", appErr.Message),
				zap.String("origin", appErr.Stack),
				zap.Error(appErr.Err),
			)
		}
		Error(c, appErr.Code, appErr.Message)
		return
	}

	// Unknown error type — log as unhandled
	if errorLogger != nil {
		errorLogger.Error("unhandled error",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Error(err),
		)
	}
	Error(c, CodeInternalServer, "internal server error")
}
