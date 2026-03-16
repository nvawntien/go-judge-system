package container

import (
	"fmt"
	"go-judge-system/pkg/config"
	"go-judge-system/services/submission/internal/adapter/inbound/http"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type App struct {
	Config   *config.Config
	Database *gorm.DB
	Router   *http.Router
	Logger   *zap.Logger
}

func NewApp(cfg *config.Config, database *gorm.DB, router *http.Router, logger *zap.Logger) *App {
	return &App{Config: cfg, Database: database, Router: router, Logger: logger}
}

func (a *App) Run() error {
	a.Router.SetupRoutes()
	port := fmt.Sprintf("%d", a.Config.Server.Port)
	a.Logger.Info("Starting Submission Service", zap.String("port", port))
	return a.Router.Start(port)
}
