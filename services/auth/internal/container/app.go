package container

import (
	"fmt"
	"go-judge-system/pkg/config"
	"go-judge-system/services/auth/internal/adapter/inbound/http"

	"go.uber.org/zap"
)

type App struct {
	Config *config.Config
	Router *http.Router
	Logger *zap.Logger
}

func NewApp(cfg *config.Config, router *http.Router, logger *zap.Logger) *App {
	return &App{
		Config: cfg,
		Router: router,
		Logger: logger,
	}
}

func (a *App) Run() error {
	a.Router.SetupRoutes()
	port := fmt.Sprintf("%d", a.Config.Server.Port)
	a.Logger.Info("Starting Auth Service", zap.String("port", port))
	return a.Router.Start(port)
}
