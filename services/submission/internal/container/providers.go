package container

import (
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/services/submission/internal/adapter/inbound/http"
	"go-judge-system/services/submission/internal/adapter/inbound/http/middleware"

	"github.com/google/wire"
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	logger.NewLogger,
)

var MiddlewareProviderSet = wire.NewSet(
	middleware.NewAuthMiddleware,
)

var InboundProviderSet = wire.NewSet(
	http.NewRouter,
)
