//go:build wireinject
// +build wireinject

package main

import (
	"go-judge-system/pkg/config"
	"go-judge-system/workers/judge/internal/container"

	"github.com/google/wire"
)

func InitializeApp(cfg *config.Config) (*container.App, func(), error) {
	wire.Build(
		container.InfrastructureProviderSet,
		container.OutboundProviderSet,
		container.InboundProviderSet,
		container.NewApp,
	)
	return nil, nil, nil
}

