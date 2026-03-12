# 📦 Go Judge System - Shared Package Library

![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go)
![Module](https://img.shields.io/badge/Module-go--judge--system%2Fpkg-4C8EDA)
![Architecture](https://img.shields.io/badge/Purpose-Shared%20Infrastructure-2F855A)

The `pkg/` module contains the shared building blocks used across services in the **Go Judge System** monorepo.

This module is intentionally kept focused on **cross-service infrastructure and utility concerns** such as configuration loading, database connections, Redis clients, logging, request/response helpers, and auth claim propagation.

---

## ✨ What This Module Provides

- **Shared configuration loading** with Viper.
- **PostgreSQL connection bootstrap** with GORM.
- **Redis client bootstrap** with connectivity validation.
- **Reusable auth claims helpers** for Gin contexts.
- **Centralized API response format** and business error mapping.
- **Structured logging setup** with Zap and log rotation.

---

## 🧱 Package Overview

### `auth/`

Utilities for working with authenticated user context inside Gin handlers and middleware.

- Defines the shared `Claims` struct.
- Stores and retrieves claims from Gin context.
- Provides convenience helpers such as `IsAdmin`, `IsSuperAdmin`, and `CanManage`.

Used by services that receive identity information from the API gateway and need consistent authorization checks.

### `cache/`

Redis bootstrap utilities.

- Builds a Redis client from typed config.
- Configures connection pool, timeouts, and DB selection.
- Performs a startup `PING` to fail fast on misconfiguration.

Used when a service needs cache, OTP state, token/session state, or temporary key-value storage.

### `config/`

Central configuration model and config loader.

- Defines shared config structs for `server`, `database`, `redis`, `logger`, `smtp`, and `jwt`.
- Loads YAML config files through Viper.
- Supports environment-variable overrides through automatic key mapping.

This package is the entry point for turning service-level `config.yaml` files into typed Go config objects.

### `database/`

PostgreSQL connection bootstrap for GORM.

- Builds DSN from typed config.
- Creates a GORM connection using the PostgreSQL driver.
- Configures connection pool sizes and lifetime.
- Verifies connectivity with an initial `Ping`.

Used by services that persist data in PostgreSQL.

### `logger/`

Structured logging bootstrap with Zap.

- Creates a logger from typed config and runtime mode.
- Writes logs to both stdout and rotating files.
- Uses console output for debug mode and JSON-friendly structure for release mode.

Used as the standard logger factory across services.

### `response/`

Shared HTTP response and error-handling layer for Gin services.

- Defines the standard API response shape.
- Maps business codes to HTTP status codes.
- Provides reusable request-binding helpers for JSON, URI params, query params, and auth claims.
- Defines `AppError` for propagating business-safe errors through use cases and handlers.

This package is especially important because it standardizes how all services return successful responses and failures.

---

## 🏗️ Design Intent

The `pkg/` module is meant for **generic, reusable, service-agnostic code**.

Good candidates for `pkg/`:
- Code reused by multiple services.
- Infrastructure bootstrap code.
- Common response/error abstractions.
- Shared auth/context helpers.

Bad candidates for `pkg/`:
- Business logic tied to a single domain.
- Service-specific DTOs, repositories, or use cases.
- HTTP handlers that only belong to one microservice.

As a rule: if the code contains domain behavior for only one service, it should stay inside that service's `internal/` tree.

---

## 📁 Directory Map

- `auth/`: Gin auth claims helpers.
- `cache/`: Redis client factory.
- `config/`: Shared config structs and loader.
- `database/`: GORM PostgreSQL connector.
- `logger/`: Zap logger factory.
- `response/`: API responses, codes, errors, and request-binding helpers.

---

## 🚀 Usage Pattern

Typical service startup flow using `pkg/` looks like this:

1. Load service config with `config.LoadConfig`.
2. Build infrastructure clients such as PostgreSQL, Redis, and logger from that config.
3. Pass those dependencies into service-specific containers and use cases.
4. Use `response` helpers in Gin handlers for consistent request binding and response formatting.
5. Use `auth` helpers in middleware and protected handlers to read/write claims.

---

## 💻 Module Dependencies

Core external libraries used by this shared module:

- `github.com/gin-gonic/gin`
- `github.com/redis/go-redis/v9`
- `github.com/spf13/viper`
- `go.uber.org/zap`
- `gopkg.in/natefinch/lumberjack.v2`
- `gorm.io/driver/postgres`
- `gorm.io/gorm`

---

## ⚠️ Notes

- `pkg/` is a standalone Go module: `go-judge-system/pkg`.
- Service-specific code should depend on this module, not the other way around.
- Changes in `pkg/response` and `pkg/config` can affect multiple services, so they should be reviewed carefully.

---
Built for the Go Judge System.