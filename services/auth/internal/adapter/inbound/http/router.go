package http

import (
	"context"
	"net/http"

	"go-judge-system/services/auth/internal/adapter/inbound/http/handler"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	pkgmiddleware "go-judge-system/pkg/middleware"
	"go-judge-system/pkg/rbac"
)

type Router struct {
	engine     *gin.Engine
	auth       *handler.AuthHandler
	user       *handler.UserHandler
	admin      *handler.AdminHandler
	middleware gin.HandlerFunc
	server     *http.Server
}

func NewRouter(authHandler *handler.AuthHandler, userHandler *handler.UserHandler, adminHandler *handler.AdminHandler, authMiddleware gin.HandlerFunc, logger *zap.Logger) *Router {
	r := gin.New()
	r.Use(pkgmiddleware.Recovery(logger))
	r.Use(pkgmiddleware.UnifiedLogger(logger))

	return &Router{
		engine:     r,
		auth:       authHandler,
		user:       userHandler,
		admin:      adminHandler,
		middleware: authMiddleware,
	}
}

func (r *Router) SetupRoutes() {
	// Health check — used by Docker HEALTHCHECK / K8s probes
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	isAdmin := pkgmiddleware.RequireRole(rbac.RoleAdmin)

	// auth api
	auth := r.engine.Group("/api/v1/auth")
	{
		auth.POST("/register", r.auth.Register.Handle)
		auth.POST("/login", r.auth.Login.Handle)
		auth.POST("/logout", r.middleware, r.auth.Logout.Handle)
		auth.POST("/logout-all", r.middleware, r.auth.LogoutAll.Handle)
		auth.POST("/refresh-token", r.auth.RefreshToken.Handle)

		email := auth.Group("/email")
		{
			email.POST("/verify", r.auth.VerifyEmail.Handle)
			email.POST("/resend-verification", r.auth.ResendVerification.Handle)
		}

		password := auth.Group("/password")
		{
			password.POST("/forgot", r.auth.ForgotPassword.Handle)
			password.POST("/reset", r.auth.ResetPassword.Handle)
			password.PUT("/change", r.middleware, r.auth.ChangePassword.Handle)
		}
	}

	me := r.engine.Group("/api/v1/me", r.middleware)
	{
		me.GET("", r.user.GetMe.Handle)
		me.PATCH("/profile", r.user.UpdateProfile.Handle)
		me.POST("/avatar", r.user.UploadAvatar.Handle)
	}

	user := r.engine.Group("/api/v1/users")
	{
		user.GET("/:username/profile", r.user.GetProfile.Handle)
	}

	// admin api
	admin := r.engine.Group("/api/v1/admin", r.middleware)
	{
		admin.PUT("/users/:user_id/role", isAdmin, r.admin.AssignRole.Handle)
	}
}

func (r *Router) Start(port string) error {
	r.server = &http.Server{
		Addr:    ":" + port,
		Handler: r.engine,
	}

	return r.server.ListenAndServe()
}

func (r *Router) Shutdown(ctx context.Context) error {
	if r.server == nil {
		return nil
	}

	return r.server.Shutdown(ctx)
}
