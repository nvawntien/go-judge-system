//go:build wireinject
// +build wireinject

package main

import (
	"go-judge-system/pkg/config"
	"go-judge-system/services/problem/internal/container"

	"github.com/google/wire"
)

func provideServerMode(cfg config.ServerConfig) string {
	return cfg.Mode
}

func InitializeApp(cfg *config.Config) (*container.App, error) {
	wire.Build(
		wire.FieldsOf(new(*config.Config), "Server", "Database", "Logger"),

		provideServerMode,

		container.InfrastructureProviderSet,
		container.OutboundProviderSet,
		container.MiddlewareProviderSet,
		container.UseCaseProviderSet,
		container.InboundProviderSet,

		container.NewApp,
	)
	return &container.App{}, nil
}
