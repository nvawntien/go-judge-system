# AGENTS.md

Go Judge System is a Go 1.25.8 multi-module online-judge backend: Auth, Problem, Submission, and Judge Worker share `pkg/`; `website/` is a separate Next.js client. Read `docs/codebase/`—especially `ARCHITECTURE.md`, `FLOWS.md`, and `DATA.md`—before changing architecture.

Key boundaries:

* Keep business rules in `internal/domain`/`internal/application`; adapters implement HTTP, gRPC, Kafka, persistence and storage behind application ports.
* Each service owns its PostgreSQL database (`auth_db`, `problem_db`, `submission_db`); use APIs/events, never another service's tables.
* Preserve the submission transaction + outbox path, Kafka contracts in `pkg/judge/`, and `AttemptID` result correlation. Do not expose official testcase inputs/expected outputs.
* Gateway JWT validation propagates `X-User-*` headers; direct service endpoints must not become an alternate untrusted auth path. Treat the privileged executor sandbox as a security boundary.
* `gateway/settings/` must stay aligned with service routers. Proto source is in `proto/`; generated Go is `pkg/pb/`.

Useful checks are module-scoped: `go test ./...`, `go vet ./...`, and `gofmt -w <changed-go-files>` from the relevant module; `docker compose config --quiet` validates local Compose. Start the complete local stack with `docker compose --profile dev --profile worker up -d --build`. No root Makefile, CI workflow, migration runner, or checked-in protobuf/Wire generation command exists—do not invent one without first establishing the toolchain.
