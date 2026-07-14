package container

import (
	"errors"
	"fmt"

	"go-judge-system/pkg/config"
	"go-judge-system/services/problem/internal/adapter/inbound/grpc"
	"go-judge-system/services/problem/internal/adapter/inbound/http"

	"go.uber.org/zap"
)

type App struct {
	Config *config.Config
	Router *http.Router
	GRPC   *grpc.Server
	Logger *zap.Logger
}

func NewApp(
	cfg *config.Config,
	router *http.Router,
	grpcServer *grpc.Server,
	logger *zap.Logger,
) *App {
	return &App{Config: cfg, Router: router, GRPC: grpcServer, Logger: logger}
}

func (a *App) Run() error {
	a.Router.SetupRoutes()

	httpPort := fmt.Sprintf("%d", a.Config.Server.Port)
	grpcPort := fmt.Sprintf("%d", a.Config.Server.GRPCPort)
	errorCh := make(chan error, 2)

	go func() {
		a.Logger.Info("Starting Problem Service HTTP server", zap.String("port", httpPort))
		if err := a.Router.Start(httpPort); err != nil {
			errorCh <- fmt.Errorf("start HTTP server: %w", err)
			return
		}
		errorCh <- errors.New("HTTP server stopped")
	}()

	go func() {
		a.Logger.Info("Starting Problem Service gRPC server", zap.String("port", grpcPort))
		if err := a.GRPC.Start(); err != nil {
			errorCh <- fmt.Errorf("start gRPC server: %w", err)
			return
		}
		errorCh <- errors.New("gRPC server stopped")
	}()

	return <-errorCh
}
