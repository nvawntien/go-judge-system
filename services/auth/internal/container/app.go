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
	authgrpc "go-judge-system/services/auth/internal/adapter/inbound/grpc"
	"go-judge-system/services/auth/internal/adapter/inbound/http"

	"go.uber.org/zap"
)

type App struct {
	Config *config.Config
	Router *http.Router
	GRPC   *authgrpc.Server
	Logger *zap.Logger
}

func NewApp(cfg *config.Config, router *http.Router, grpcServer *authgrpc.Server, logger *zap.Logger) *App {
	return &App{
		Config: cfg,
		Router: router,
		GRPC:   grpcServer,
		Logger: logger,
	}
}

func (a *App) Run() error {
	a.Router.SetupRoutes()
	httpPort := fmt.Sprintf("%d", a.Config.Server.Port)
	grpcPort := fmt.Sprintf("%d", a.Config.Server.GRPCPort)

	httpErrCh := make(chan error, 1)
	grpcErrCh := make(chan error, 1)
	go func() {
		a.Logger.Info("Starting Auth Service HTTP server", zap.String("port", httpPort))
		httpErrCh <- a.Router.Start(httpPort)
	}()
	go func() {
		a.Logger.Info("Starting Auth Service gRPC server", zap.String("port", grpcPort))
		grpcErrCh <- a.GRPC.Start()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case err := <-httpErrCh:
		a.GRPC.Stop()
		grpcErr := <-grpcErrCh
		if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			return errors.Join(fmt.Errorf("start HTTP server: %w", err), grpcErr)
		}
		return grpcErr
	case err := <-grpcErrCh:
		if err == nil {
			err = errors.New("gRPC server stopped")
		}
		return errors.Join(fmt.Errorf("start gRPC server: %w", err), a.shutdownHTTP(httpErrCh))
	case sig := <-signalCh:
		a.Logger.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a.GRPC.Stop()
	return errors.Join(a.Router.Shutdown(ctx), a.waitForHTTP(httpErrCh), a.waitForGRPC(grpcErrCh))
}

func (a *App) shutdownHTTP(httpErrCh <-chan error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a.GRPC.Stop()
	return errors.Join(a.Router.Shutdown(ctx), a.waitForHTTP(httpErrCh))
}

func (a *App) waitForHTTP(httpErrCh <-chan error) error {
	err := <-httpErrCh
	if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
		return fmt.Errorf("stop HTTP server: %w", err)
	}
	return nil
}

func (a *App) waitForGRPC(grpcErrCh <-chan error) error {
	err := <-grpcErrCh
	if err != nil {
		return fmt.Errorf("stop gRPC server: %w", err)
	}
	return nil
}
