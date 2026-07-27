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

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- a.Router.Start(port)
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case err := <-serverErrCh:
		if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			return err
		}
		return nil
	case sig := <-signalCh:
		a.Logger.Info("shutdown signal received", zap.String("signal", sig.String()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.Router.Shutdown(ctx); err != nil {
		return err
	}

	err := <-serverErrCh
	if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
		return err
	}

	return nil
}
