package container

import (
	"context"
	"fmt"

	"go-judge-system/pkg/config"
	"go-judge-system/services/problem/internal/adapter/inbound/http"
	testcase "go-judge-system/services/problem/internal/application/usecase/test_case"

	"go.uber.org/zap"
)

type App struct {
	Config   *config.Config
	Router   *http.Router
	Logger   *zap.Logger
	GCRunner *testcase.GCRunner
}

func NewApp(cfg *config.Config, router *http.Router, logger *zap.Logger, gcRunner *testcase.GCRunner) *App {
	return &App{Config: cfg, Router: router, Logger: logger, GCRunner: gcRunner}
}

func (a *App) Run() error {
	a.Router.SetupRoutes()

	// Start background GC goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.GCRunner.Start(ctx)

	port := fmt.Sprintf("%d", a.Config.Server.Port)
	a.Logger.Info("Starting Problem Service", zap.String("port", port))
	return a.Router.Start(port)
}
