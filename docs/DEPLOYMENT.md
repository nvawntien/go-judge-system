# AstraCode dual-node production deployment

Production uses immutable GHCR release images, Docker Compose, and a protected
manual GitHub Actions deployment. It has two distinct nodes:

```text
GitHub Actions runner
  ├─ SSH ────────────────────────────────> App Node (astracode-app)
  └─ SSH ProxyJump through App Node ─────> Judge Node (astracode-judge-01)
```

This document describes the production deployment contract. It does not claim
zero downtime or replace host/sandbox security review.

## Release policy

Only a published semantic release tag is valid for production:

```text
vX.Y.Z
vX.Y.Z-prerelease
```

The image workflow publishes all custom images for semantic releases with both
the release tag and `sha-<full-commit>`:

```text
website
auth-service
problem-service
submission-service
judge-worker
sandbox
```

SHA images are useful for CI and staging, but production deployment accepts
only `vX.Y.Z` tags. It never uses `latest`, a branch name, or an arbitrary
commit. Protect release Git tags and configure registry policy so a release tag
cannot be silently overwritten.

## Node ownership boundary

GitHub Actions uploads repository-controlled files only. The following files
are node-owned operational inputs and must never be overwritten, copied from
the repository, or edited by a deployment script:

```text
# App Node
/etc/astracode/app-node.override.yml
/etc/astracode/stack.env
/etc/astracode/service.env
/etc/astracode/postgres.env
/etc/astracode/redis.env
/etc/astracode/symmetric.json

# Judge Node
/etc/astracode/judge-node.compose.yml
/etc/astracode/judge/config.yaml
```

No database, Kafka, Redis, MinIO, JWT, SMTP, WireGuard, or sandbox secret is a
GitHub Actions secret or deployment-bundle file.

## App Node

The App Node owns these services:

```text
website
envoy
gateway
postgres
redis
minio
kafka
kafka-init
auth-service
problem-service
submission-service
```

It does **not** run production `go-judge` or `judge-worker`.

Every App deployment must compose exactly these files:

```text
/opt/astracode/docker-compose.yml
/opt/astracode/docker-compose.prod.yml
/etc/astracode/app-node.override.yml
```

The external override keeps Judge-only services behind an opt-in `judge`
profile and supplies the private network bindings required by the Judge Node:

```text
Problem gRPC: 10.77.0.1:9092
Kafka:        10.77.0.1:9094
MinIO:        10.77.0.1:9000
```

`scripts/deploy-app-node.sh` validates the complete three-file Compose model,
pulls and recreates only App services, runs edge smoke checks, preserves the
public avatar bucket behavior, and records state under:

```text
/opt/astracode/.deploy/app/current-image-tag
/opt/astracode/.deploy/app/previous-image-tag
```

`kafka-init` is one-shot and is intentionally not treated as a permanently
running health-checked service.

Kafka runs with a fixed 512 MiB JVM heap below its 1 GiB container memory
limit. Its recurring Docker healthcheck is a lightweight local TCP probe;
`kafka-init` separately performs bounded Kafka protocol-readiness retries before
the idempotent topic-creation commands. App activation starts Kafka, forcibly
recreates and runs the `kafka-init` Compose service to successful completion,
and only then activates the complete App service set.

### One-time benchmark account provisioning

The Auth image contains `/app/provision-benchmark-users` for the fixed
`benchmark_judge_001` through `benchmark_judge_10000` fixture pool. It does not
create sessions, tokens, mail, or verification artifacts. Operators must use
the exact immutable Auth image tag already deployed to the App Node, first run
a dry-run, review its sanitized target/range and exact confirmation phrase,
then run the same command with `--apply --confirm '<printed phrase>'`.
Verify that deployed immutable Auth tag before either invocation; never run a
local, `develop`, or `latest` image against the production Auth database.

Use an interactive TTY so the command can request the shared benchmark password
without echoing it; do not put a password in a command line or environment.
For example, with the same App Compose files, project name, and image tag used
by the deployed stack, run the Auth service as a one-shot no-dependency command
with its entrypoint overridden to `/app/provision-benchmark-users`. This does
not restart the running Auth service. A password-file alternative is accepted
only when it is a regular, non-symlink owner-only file readable by the runtime
`appuser`; verify bind-mount ownership before relying on it.

```bash
# First run dry-run and review the printed target/range confirmation phrase.
docker compose --project-name go-judge-system <same-app-compose-arguments> \
  run --rm --no-deps -it --entrypoint /app/provision-benchmark-users auth-service \
  --config-dir /app/config --start 1 --count 50

# Then use the exact phrase printed above; the password is prompted without echo.
docker compose --project-name go-judge-system <same-app-compose-arguments> \
  run --rm --no-deps -it --entrypoint /app/provision-benchmark-users auth-service \
  --config-dir /app/config --start 1 --count 50 --apply --confirm '<printed phrase>'
```

Normal provisioning is idempotent: canonical existing benchmark accounts are
skipped only when their stored hash matches the supplied password. Normal mode
never resets, deletes, or changes existing accounts; conflicts stop it safely.
The fixed range is `1..10000`; `%03d` keeps `001..100` unchanged and naturally
extends the deterministic namespace through `10000`. For a 10K fixture build,
plan the complete range first (`--start 1 --count 10000`), then apply only the
printed exact confirmation. The command reuses one database pool, processes
accounts sequentially with the production password encoder, and prints only
count-based progress every 500 accounts—never passwords or hashes.

### Benchmark password rotation

Rotation is a separate, explicit fixture-only operation for an already-created
canonical range. It never creates users, accepts no interactive password, and
updates only passwords in one transaction after validating the entire range.

1. Create one temporary local file containing the **new benchmark password**;
   run `chmod 600` on it. Do not place the password itself in documentation,
   command examples, shell history, logs, commits, or chat.
2. Copy that same owner-only file to the App Node using a secure operator
   transfer such as `scp`; ensure the container `appuser` can read the mounted
   file without weakening its permissions.
3. Run the explicit rotation dry-run and copy its printed `ROTATE PASSWORD ...`
   confirmation phrase.
4. Run apply with `--password-file` and that exact phrase.
5. Optionally verify one benchmark account with the same file, then use the
   original file with `bootstrap-sessions` to create fresh local sessions.
6. Delete temporary plaintext password files after bootstrap succeeds.

```bash
# This makes no changes and prints a target/range-bound ROTATE PASSWORD phrase.
docker compose --project-name go-judge-system <same-app-compose-arguments> \
  run --rm --no-deps -it --entrypoint /app/provision-benchmark-users auth-service \
  --config-dir /app/config --start 51 --count 50 --rotate-password --dry-run

# Use the same exact immutable Auth image tag as the running service. The
# password itself is never placed in the command line or environment.
docker compose --project-name go-judge-system <same-app-compose-arguments> \
  run --rm --no-deps -it --entrypoint /app/provision-benchmark-users auth-service \
  --config-dir /app/config --start 51 --count 50 --rotate-password --apply \
  --password-file /secure-mounted/new-benchmark-password \
  --confirm 'ROTATE PASSWORD benchmark_judge_051..benchmark_judge_100 ON <printed-target>'
```

Do not use this rotation mode for normal accounts. A missing, inactive,
suspended, role-changed, or otherwise noncanonical fixture makes the full range
fail before any password update is committed.

Rotation does **not** revoke already-issued sessions. Existing access and
refresh sessions can remain valid according to the normal Auth/session
lifecycle; bootstrap fresh benchmark sessions after rotation.

## Judge Node

The private Judge Node runs only:

```text
judge_sandbox (Compose service: go-judge)
judge_worker  (Compose service: judge-worker)
```

It reaches App services through the private network/WireGuard. Its Compose and
runtime configuration are external and node-owned:

```text
/etc/astracode/judge-node.compose.yml
/etc/astracode/judge/config.yaml
```

`scripts/deploy-judge-node.sh` does not edit either file. For each release it
creates a secure temporary Compose override containing only these two image
fields:

```yaml
services:
  go-judge:
    image: ghcr.io/nvawntien/go-judge-system/sandbox:vX.Y.Z
  judge-worker:
    image: ghcr.io/nvawntien/go-judge-system/judge-worker:vX.Y.Z
```

It then composes the node-owned file plus the temporary image override, pulls
and recreates only `go-judge` and `judge-worker`, verifies both containers,
checks sandbox HTTP reachability without publishing a port, and confirms the
Worker listens on the `server.grpc_port` derived from its external config.

Judge state is recorded independently:

```text
/opt/astracode/.deploy/judge/current-image-tag
/opt/astracode/.deploy/judge/previous-image-tag
```

Restarting a Worker can cause a Kafka consumer-group rebalance. The scripts use
normal Compose recreation and no `kill -9`, global prune, volume deletion, or
invented queue-draining mechanism.

## GitHub Environment secrets

The `production` Environment requires pinned SSH material for both hosts:

```text
PROD_APP_HOST
PROD_APP_PORT
PROD_APP_USER
PROD_APP_SSH_KEY

PROD_JUDGE_HOST
PROD_JUDGE_PORT
PROD_JUDGE_USER
PROD_JUDGE_SSH_KEY

PROD_KNOWN_HOSTS
```

`PROD_KNOWN_HOSTS` must contain pinned host keys for the App Node and the Judge
Node. The workflow creates an ephemeral SSH configuration with
`ProxyJump astracode-app`, `BatchMode yes`, `IdentitiesOnly yes`, and strict
host-key checking. It never disables host verification.

If one key is deliberately authorized on both hosts, its value may be supplied
to both logical key secrets; the workflow does not assume that arrangement.

## Deployment sequence

The manual **Deploy production** workflow:

1. validates and checks out the requested semantic release tag;
2. checks that all six GHCR release image manifests exist;
3. uploads non-secret bundles before mutating either node;
4. deploys and verifies the Judge Node first;
5. deploys and verifies the App Node second.

Judge-first ordering ensures the new App release is not made live before its
compatible Worker/Sandbox release is ready.

## Rollback

Each node can be rolled back independently:

```bash
# Run on the relevant node.
/opt/astracode/scripts/deploy-app-node.sh --rollback
/opt/astracode/scripts/deploy-judge-node.sh --rollback
```

On local activation failure, each script attempts its own previous valid
semantic release and still exits nonzero to report that the requested release
failed.

App rollback changes the AstraCode application image tag while reusing the
currently deployed Compose bundle and node-owned override. It does not restore
an earlier base Compose infrastructure configuration, including prior Kafka
heap, resource-limit, or healthcheck settings; those require an explicit bundle
rollback by an operator when necessary.

If Judge deployment fails, the workflow stops before touching App. If Judge
succeeds but App deployment fails, the App script performs its own rollback and
the workflow explicitly requests Judge rollback. A failed Judge rollback is a
visible workflow failure requiring operator intervention.

This is orchestration, not a distributed transaction. Review schema and
protocol compatibility before release; application rollback is not database
rollback, and GORM `AutoMigrate` can make older application images
incompatible with newer schema.

## One-time server preparation

Operators must provision and permission these pre-existing paths before the
first deployment:

```text
/opt/astracode
/opt/astracode/.deploy/app
/opt/astracode/.deploy/judge
/etc/astracode/*                 # App Node ownership
/etc/astracode/judge/*           # Judge Node ownership
```

The deployment accounts need Docker access, read access to their node-owned
configuration, write access to `/opt/astracode`, and no broader privileges than
required. Private network routes and firewall rules must permit only the
intended Judge-to-App interfaces; do not publish Kafka, Problem gRPC, MinIO,
Worker gRPC, or Sandbox HTTP to the public Internet.

## Deprecated command

`scripts/deploy-production.sh` intentionally exits without deploying. It is a
deprecated combined-node entrypoint retained only to prevent accidental use;
use the node-specific scripts or the GitHub workflow instead.
