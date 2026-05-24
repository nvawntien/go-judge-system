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
func SetErrorLogger(l *zap.Logger) {
    errorLogger = l
}

// HandleVoid: Used for APIs that receive JSON body data, but do not return data.
func HandleVoid[Req any](c *gin.Context, fn func(context.Context, Req) error, successCode int, successMsg string) {
    var req Req
    if err := c.ShouldBindJSON(&req); err != nil {
        HandleError(c, NewAppError(CodeBadRequest, "invalid request payload", err))
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
        HandleError(c, NewAppError(CodeUnauthorized, "unauthorized", nil))
        return
    }

    var req Req
    if err := c.ShouldBindJSON(&req); err != nil {
        HandleError(c, NewAppError(CodeBadRequest, "invalid request payload", err))
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
        HandleError(c, NewAppError(CodeParamInvalid, "invalid uri params", err))
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
        HandleError(c, NewAppError(CodeBadRequest, "invalid request payload", err))
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
        HandleError(c, NewAppError(CodeUnauthorized, "unauthorized", nil))
        return
    }

    var req Req
    if err := c.ShouldBindJSON(&req); err != nil {
        HandleError(c, NewAppError(CodeBadRequest, "invalid request payload", err))
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
        HandleError(c, NewAppError(CodeUnauthorized, "unauthorized", nil))
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
        HandleError(c, NewAppError(CodeParamInvalid, "invalid uri params", err))
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
        HandleError(c, NewAppError(CodeBadRequest, "invalid request payload", err))
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
        HandleError(c, NewAppError(CodeUnauthorized, "unauthorized", nil))
        return
    }

    var req Req
    if err := c.ShouldBindUri(&req); err != nil {
        HandleError(c, NewAppError(CodeParamInvalid, "invalid uri params", err))
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
        HandleError(c, NewAppError(CodeUnauthorized, "unauthorized", nil))
        return
    }

    var req Req
    if err := c.ShouldBindUri(&req); err != nil {
        HandleError(c, NewAppError(CodeParamInvalid, "invalid uri params", err))
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
        HandleError(c, NewAppError(CodeBadRequest, "invalid query parameters", err))
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
        HandleError(c, NewAppError(CodeParamInvalid, "invalid uri params", err))
        return
    }

    var query Q
    if err := c.ShouldBindQuery(&query); err != nil {
        HandleError(c, NewAppError(CodeBadRequest, "invalid query parameters", err))
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
        HandleError(c, NewAppError(CodeUnauthorized, "unauthorized", nil))
        return
    }

    var req Req
    if err := c.ShouldBindQuery(&req); err != nil {
        HandleError(c, NewAppError(CodeBadRequest, "invalid query parameters", err))
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
func HandleVoidWithParamsAndBody[P any, B any](c *gin.Context, fn func(context.Context, P, B) error, successCode int, successMsg string) {
    var params P
    if err := c.ShouldBindUri(&params); err != nil {
        HandleError(c, NewAppError(CodeParamInvalid, "invalid uri params", err))
        return
    }

    var body B
    if err := c.ShouldBindJSON(&body); err != nil {
        HandleError(c, NewAppError(CodeBadRequest, "invalid request payload", err))
        return
    }

    if err := fn(c.Request.Context(), params, body); err != nil {
        HandleError(c, err)
        return
    }

    SuccessWithMessage(c, successCode, successMsg, nil)
}

// HandleWithParamsAndBody: URI params + JSON body + claims → data. Used for Create TestCase (problem ID + body).
func HandleWithParamsAndBody[P any, B any, Res any](c *gin.Context, fn func(context.Context, P, B) (Res, error), successCode int) {
    var params P
    if err := c.ShouldBindUri(&params); err != nil {
        HandleError(c, NewAppError(CodeParamInvalid, "invalid uri params", err))
        return
    }

    var body B
    if err := c.ShouldBindJSON(&body); err != nil {
        HandleError(c, NewAppError(CodeBadRequest, "invalid request payload", err))
        return
    }

    res, err := fn(c.Request.Context(), params, body)
    if err != nil {
        HandleError(c, err)
        return
    }

    Success(c, successCode, res)
}

// HandleWithParamsAndForm: URI params + multipart form/form-data + claims → data.
func HandleWithParamsAndForm[P any, F any, Res any](c *gin.Context, fn func(context.Context, P, F) (Res, error), successCode int) {
    var params P
    if err := c.ShouldBindUri(&params); err != nil {
        HandleError(c, NewAppError(CodeParamInvalid, "invalid uri params", err))
        return
    }

    var form F
    if err := c.ShouldBind(&form); err != nil {
        HandleError(c, NewAppError(CodeBadRequest, "invalid form data", err))
        return
    }

    res, err := fn(c.Request.Context(), params, form)
    if err != nil {
        HandleError(c, err)
        return
    }

    Success(c, successCode, res)
}

// HandleError: Handles errors, logs them to the context for middleware, and returns HTTP responses.
func HandleError(c *gin.Context, err error) {
    // 1. LOG ERROR (including stack trace if it's an AppError)
    if err != nil {
        c.Error(err)
    }

    var appErr *AppError
    if errors.As(err, &appErr) {
        Error(c, appErr.Code, appErr.Message)
        return
    }

    Error(c, CodeInternalServer, "internal server error")
}