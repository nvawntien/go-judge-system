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
	// Health check — used by Docker HEALTHCHECK / K8s probes
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.engine.Group("/api/v1/auth")
	{
		v1.POST("/register", r.authHandler.RegisterHandler.Handle)
		v1.POST("/login", r.authHandler.LoginHandler.Handle)
		v1.POST("/refresh-token", r.authHandler.RefreshTokenHandler.Handle)

		v1.POST("/verify-acti", r.authHandler.VerifyActivationHandler.Handle)
		v1.POST("/resend-otp", r.authHandler.ResendOTPHandler.Handle)

		v1.POST("password/forgot", r.authHandler.ForgotPasswordHandler.Handle)
		v1.POST("password/reset", r.authHandler.ResetPasswordHandler.Handle)

		v1.POST("/email/verify", r.authHandler.VerifyEmailHandler.Handle)
		v1.POST("email/resend-verification", r.authHandler.ResendVerificationEmailHandler.Handle)
	}

	// Authenticated routes
	authenticated := v1.Group("")
	authenticated.Use(r.authMiddleware)
	{
		authenticated.PUT("/password/change", r.authHandler.ChangePasswordHandler.Handle)
		authenticated.POST("/logout", r.authHandler.LogoutHandler.Handle)
	}

	// Super Admin routes
	admin := v1.Group("/admin")
	admin.Use(r.authMiddleware)
	{
		admin.PUT("/:username/role", r.authHandler.UpdateUserRoleHandler.Handle)
	}
}

func (r *Router) Start(port string) error {
	return r.engine.Run(":" + port)
}
