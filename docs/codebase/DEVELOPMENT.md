# Development workflow

## Prerequisites and configuration

Use Go 1.25.8 (the workspace and Docker builders declare it), Docker Engine/Compose, and Node/npm for `website/`. Runtime configs live in `services/*/config/config.yaml` and `workers/judge/config/config.yaml`; Compose reads `environment/postgres.env`, `environment/redis.env`, and `environment/service.env`. Do not commit real secrets. Viper maps dotted config paths to uppercase underscore environment names, such as `DATABASE_PASSWORD` and `JWT_ACCESS_SECRET`.

## Run the full stack

From repository root, the checked-in quick start is:

```bash
docker compose --profile dev --profile worker up -d --build
docker compose ps
docker compose down
```

The worker is profile-gated; omit `--profile worker` to start the API/infrastructure without it. Envoy exposes the gateway at `http://localhost:8080`; MailHog and Kafka UI are dev-profile services. Compose creates databases and Kafka topics automatically through the mounted bootstrap assets.

## Run a Go component locally

Each runtime is a separate Go module but `go.work` links them. From its module directory:

```bash
go run ./cmd/server
go test ./...
go vet ./...
```

The worker accepts `-config` (default `/app/config`), so local usage needs a config directory reachable by that flag. The other Go mains currently load `/app/config` unconditionally; use Docker/Compose or supply that expected path in a local environment. This limitation is derived from their `main.go` files.

## Build, test and quality checks

There is no root Makefile or CI configuration in this checkout. Use module-scoped commands. Tests are colocated with source (`*_test.go`) across shared packages, services, and worker adapters/use cases. A safe non-writing discovery/build sequence is:

```bash
go test ./pkg/...
(cd services/auth && go test ./...)
(cd services/problem && go test ./...)
(cd services/submission && go test ./...)
(cd workers/judge && go test ./...)
docker compose config --quiet
```

Format modified Go files with `gofmt -w <files>`. Wire injectors are in each `cmd/server/wire.go` and generated output is checked in as `wire_gen.go`; no Wire generation command or `buf`/`protoc` configuration was found, so do not guess a generation command. Proto sources are `proto/problem/v1/problem.proto` and `proto/judge/v1/run.proto`; generated Go is checked in under `pkg/pb/`.

For website development, inspect `website/package.json`; its defined script is `npm run build` (`next build`). `node_modules` and `.next` are generated local artifacts and should not be used as architecture evidence.

## Migrations and operational checks

There is no migration runner. Startup repository constructors call GORM `AutoMigrate`, and `infra/postgres/init.sql` only creates the three databases. Before introducing a migration tool or altering schemas, consult [DATA.md](DATA.md) and preserve service database ownership. Validate gateway configuration with the Compose-provided KrakenD image when infrastructure is running; do not assume all service-local routes are gateway-exposed.
