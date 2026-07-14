package container

import (
	"context"
	"errors"
	"fmt"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	httpErrCh := make(chan error, 1)
	grpcErrCh := make(chan error, 1)

	go func() {
		a.Logger.Info("Starting Problem Service HTTP server", zap.String("port", httpPort))
		httpErrCh <- a.Router.Start(httpPort)
	}()

	go func() {
		a.Logger.Info("Starting Problem Service gRPC server", zap.String("port", grpcPort))
		grpcErrCh <- a.GRPC.Start()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case err := <-httpErrCh:
		if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			return fmt.Errorf("start HTTP server: %w", err)
		}
		return nil
	case err := <-grpcErrCh:
		if err == nil {
			err = errors.New("gRPC server stopped")
		}
		return errors.Join(fmt.Errorf("start gRPC server: %w", err), a.shutdownHTTP(httpErrCh))
	case sig := <-signalCh:
		a.Logger.Info("shutdown signal received", zap.String("signal", sig.String()))
		return a.shutdownHTTP(httpErrCh)
	}
}

func (a *App) shutdownHTTP(serverErrCh <-chan error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shutdownErr := a.Router.Shutdown(ctx)
	serverErr := <-serverErrCh
	if serverErr != nil && !errors.Is(serverErr, nethttp.ErrServerClosed) {
		serverErr = fmt.Errorf("stop HTTP server: %w", serverErr)
	} else {
		serverErr = nil
	}

	return errors.Join(shutdownErr, serverErr)
}
