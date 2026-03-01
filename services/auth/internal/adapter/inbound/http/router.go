package http

import (
	"go-judge-system/services/auth/internal/adapter/inbound/http/handler"

	"github.com/gin-gonic/gin"
)

type Router struct {
	engine         *gin.Engine
	authHandler    *handler.AuthHandler
	authMiddleware gin.HandlerFunc
}

func NewRouter(authHandler *handler.AuthHandler, authMiddleware gin.HandlerFunc) *Router {
	r := gin.Default()
	return &Router{
		engine:         r,
		authHandler:    authHandler,
		authMiddleware: authMiddleware,
	}
}

func (r *Router) SetupRoutes() {
	v1 := r.engine.Group("/api/v1/auth")
	{
		v1.POST("/register", r.authHandler.RegisterHandler.Handle)
		v1.POST("/verify-activation", r.authHandler.VerifyActivationHandler.Handle)
		v1.POST("/resend-otp", r.authHandler.ResendOTPHandler.Handle)

		v1.POST("/forgot-password", r.authHandler.ForgotPasswordHandler.Handle)
		v1.POST("/verify-forgot-password", r.authHandler.VerifyForgotPasswordHandler.Handle)
		v1.POST("/reset-password", r.authHandler.ResetPasswordHandler.Handle)

		v1.POST("/login", r.authHandler.LoginHandler.Handle)
	}

	// Authenticated routes
	authenticated := v1.Group("")
	authenticated.Use(r.authMiddleware)
	{
		authenticated.PUT("/change-password", r.authHandler.ChangePasswordHandler.Handle)
	}
}

func (r *Router) Start(port string) error {
	return r.engine.Run(":" + port)
}
