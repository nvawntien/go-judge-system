# Contributing to AstraCode

Thank you for helping improve AstraCode. This repository combines several Go
services, a Next.js frontend, shared contracts, and operational configuration,
so small, well-explained changes are the easiest to review safely.

By participating, you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).
Report suspected vulnerabilities through the private process in
[SECURITY.md](SECURITY.md), not through a normal public issue.

## Getting started

1. Fork the repository and clone your fork.
2. Add this repository as an `upstream` remote.
3. Update your local `develop` branch before starting normal development work.
4. Create a short-lived, scoped branch from `develop`.

For example:

```bash
git clone https://github.com/<your-account>/go-judge-system.git
cd go-judge-system
git remote add upstream https://github.com/nvawntien/go-judge-system.git
git fetch upstream
git switch -c feat/short-description upstream/develop
```

Do not work directly on `main`. Keep a pull request focused on one problem and
avoid unrelated formatting, dependency, or architectural changes.

## Branching model

Normal changes follow this flow:

```text
feat/... | fix/... | docs/... | chore/...
                    ↓
                 develop
                    ↓
              main for releases
```

The prefixes are repository conventions, not a claim that every branch rule is
automatically enforced. Normal pull requests should target `develop`; release
promotion to `main` is handled separately by maintainers.

## Development architecture

AstraCode is a Go 1.25.8 workspace with five modules and a separate Next.js
frontend:

- `website/` — Next.js user interface;
- `services/auth/` — identities, sessions, profiles, and roles;
- `services/problem/` — problems, tags, authoring, and testcase metadata;
- `services/submission/` — submissions, attempts, results, outbox, and SSE;
- `workers/judge/` — asynchronous judging and interactive Run Code;
- `pkg/` — shared Go packages and generated service contracts;
- `proto/` — source protocol definitions;
- `gateway/` — KrakenD routes, JWT validation, and API composition; and
- `envoy/` — public edge routing.

Respect the existing domain, application, port, and adapter boundaries. Auth,
Problem, and Submission each own their database; never read or write another
service's tables directly. Use the established APIs, gRPC contracts, and Kafka
events for cross-service communication. Keep official testcase inputs and
expected outputs confidential, and treat the executor as a security boundary.

Start with the [architecture](docs/codebase/ARCHITECTURE.md), [execution
flows](docs/codebase/FLOWS.md), and [data ownership](docs/codebase/DATA.md)
documentation before changing service boundaries.

## Local setup and validation

Follow the [README Quick start](README.md#quick-start) for the local stack. Go
checks are module-scoped; there is no single root-module `go test ./...`
substitute for all five modules.

Run the checks relevant to your change. The full CI-equivalent Go and frontend
set is:

```bash
for module in pkg services/auth services/problem services/submission workers/judge; do
  (cd "$module" && go test ./... && go vet ./... && go test -race ./...)
done

(cd website && npm ci && npm run typecheck && npm run build)
```

Validate the development Compose model without relying on local secret files:

```bash
POSTGRES_ENV_FILE=./environment/postgres.env.example \
REDIS_ENV_FILE=./environment/redis.env.example \
SERVICE_ENV_FILE=./environment/service.env.example \
GATEWAY_JWK_FILE=./gateway/symmetric.json \
docker compose config --quiet
```

Format changed Go files with `gofmt`. Keep gateway endpoint definitions aligned
with service routers when changing public HTTP contracts. Do not guess protobuf
or Wire generation commands; establish the repository toolchain in the pull
request when generated artifacts must change.

## Database and schema changes

Each service owns its PostgreSQL database. The current code uses GORM
`AutoMigrate` at service startup and does not include a versioned migration
runner. A schema pull request must therefore describe data impact, backward and
forward compatibility, deployment ordering, and rollback limitations. Do not
add cross-service table access as a shortcut.

## Pull requests

In the pull request description, explain:

- the problem or use case;
- the proposed solution and scope;
- the tests and checks actually run;
- API, schema, configuration, and backward-compatibility effects;
- deployment or operational implications, when relevant; and
- screenshots for meaningful user-interface changes.

Call out checks you could not run and explain why. Reviewers should not have to
infer migrations, generated-file changes, or production impact from the diff.

## Security and sensitive data

Never commit `.env` files containing secrets, production credentials, private
keys, tokens, private testcase data, or node-owned production configuration.
Redact secrets and personal data from logs, screenshots, fixtures, and issue
reports. Do not disclose a suspected vulnerability through a regular GitHub
issue; follow [SECURITY.md](SECURITY.md).

## Commit quality

Prefer clear, scoped commits that leave the repository in a reviewable state.
Prefixes such as `feat:`, `fix:`, `docs:`, and `chore:` are useful conventions,
but the clarity and accuracy of the message matter more than a rigid format.
