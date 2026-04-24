package middleware

import (
	"errors"
	"go-judge-system/pkg/response"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func UnifiedLogger(logger *zap.Logger) gin.HandlerFunc {
	sugar := logger.WithOptions(zap.WithCaller(false)).Sugar()

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		if path == "/health" {
			return
		}

		status := c.Writer.Status()
		latency := time.Since(start)

		userTag := ""
		if userID := c.GetHeader("X-User-ID"); userID != "" {
			userTag = "[User:" + userID + "] "
		}

		coloredMethod := colorMethod(method)

		if len(c.Errors) > 0 {
			err := c.Errors[0].Err
			var appErr *response.AppError

			if errors.As(err, &appErr) {
				if status >= 500 {
					sugar.Errorf("%s %-25s | %3d | %s\n    ↳ ROOT:   %v\n    ↳ ORIGIN: %s",
						coloredMethod, path, status, userTag, appErr.Err, appErr.Stack)
				} else {
					sugar.Warnf("%s %-25s | %3d | %s\n    ↳ ROOT:   %v",
						coloredMethod, path, status, userTag, appErr.Err)
				}
			} else {
				if status >= 500 {
					sugar.Errorf("%s %-25s | %3d | %s\n    ↳ UNHANDLED: %v",
						coloredMethod, path, status, userTag, err)
				} else {
					sugar.Warnf("%s %-25s | %3d | %s\n    ↳ UNHANDLED: %v",
						coloredMethod, path, status, userTag, err)
				}
			}
			return
		}

		sugar.Infof("%s %-25s | %3d | %s%v", coloredMethod, path, status, userTag, latency)
	}
}

func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					zap.Any("error", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("stack", string(debug.Stack())),
				)
				response.Error(c, response.CodeInternalServer, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}

func colorMethod(method string) string {
	padded := method
	for len(padded) < 6 {
		padded += " "
	}

	switch method {
	case "GET":
		return "\033[34m" + padded + "\033[0m"
	case "POST":
		return "\033[32m" + padded + "\033[0m"
	case "PUT":
		return "\033[33m" + padded + "\033[0m"
	case "DELETE":
		return "\033[31m" + padded + "\033[0m" 
	case "PATCH":
		return "\033[36m" + padded + "\033[0m" 
	default:
		return padded 
	}
}