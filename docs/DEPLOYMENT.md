# AstraCode production deployment

This document describes the reference v0.1.0 deployment on a single Linux VPS.
It uses immutable images from GitHub Container Registry (GHCR), Docker Compose,
and a host-managed TLS reverse proxy. It is intentionally separate from the
local development workflow.

The reference topology is production-oriented, but it is not a claim of
Kubernetes-style zero downtime or a complete hardening program. AstraCode
executes untrusted code; operators must review the executor, host, and network
boundary before serving public traffic.

> **Judge boundary:** the shipped executor image needs privileged Linux
> namespace/cgroup operations. The production overlay removes the executor from
> the general application network and does not pass service secrets to the
> Worker, but those are defence-in-depth measures—not a same-host containment
> guarantee. Run public untrusted submissions on a dedicated Judge node (or an
> equivalently isolated executor host) rather than alongside Auth and stateful
> services on the application VPS.

## Release and deployment model

```text
pull request -> CI -> develop -> main -> vX.Y.Z tag
                                         |
                                         v
                              immutable GHCR images
                                         |
                             manual production approval
                                         |
                                         v
                    /opt/astracode on the production VPS
```

- `.github/workflows/ci.yml` validates Go modules, the website, Compose, and all
  custom images. It has read-only repository permission and no production
  secrets.
- `.github/workflows/release-images.yml` runs for a semantic release tag and
  publishes every custom image with both `vX.Y.Z` and `sha-<full-commit>` tags.
- `.github/workflows/deploy-production.yml` is manual, uses the GitHub
  `production` Environment, uploads only non-secret deployment material, and
  activates an immutable tag.
- Application secrets remain in `/etc/astracode` on the VPS. They are not sent
  on every deployment.

Normal deployments pull images. They do not compile Go or Next.js on the VPS.

## Production topology

The Compose overlay adds the standalone Next.js website and makes Envoy the
only host-bound AstraCode container:

```text
TLS reverse proxy on host
          |
          v
127.0.0.1:8080 -> Envoy
                    |-- /              -> Next.js website
                    |-- /api/*         -> KrakenD -> Go services
                    |-- /events/*      -> Submission Service SSE
                    `-- /avatars/*     -> MinIO avatar bucket
```

PostgreSQL, Redis, Kafka, MinIO, KrakenD, Go services, Judge Worker, and the
executor are reachable only on the Compose network. The production Envoy also
removes incoming `X-User-ID`, `X-Role`, `X-Username`, and `X-Token-Iat`
headers before KrakenD derives trusted identity claims.

Named volumes persist PostgreSQL, Redis, Kafka, MinIO, and testcase cache data.
MailHog and Kafka UI remain development-profile services and do not start in
the production topology.

### Public Judge placement

For public submissions, keep `judge-worker` and `go-judge` on a dedicated
Judge node. The application node continues to own Auth, Problem, Submission,
Kafka, and persistent data; the Judge node consumes jobs and publishes results
through Kafka, calls Problem's existing worker gRPC API for short-lived testcase
metadata, and downloads the resulting presigned testcase bundle. Do not add
direct database access from the Judge node.

Connect the nodes through a private network, VPN, or equivalent authenticated
and encrypted transport. Do not expose Kafka, Problem gRPC, the Worker gRPC
port, or the executor HTTP API publicly. The single-VPS Compose topology
remains suitable for local development and controlled testing, not approval to
co-locate a privileged public-code executor with application secrets and data.

## Production email

Auth sends account-verification and password-reset messages through a generic
SMTP adapter. Configure any standards-compliant transactional SMTP provider in
the VPS-only `/etc/astracode/service.env`; no provider SDK or credentials are
part of the image, Compose files, or GitHub Actions.

Required variables are `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`,
`SMTP_PASSWORD`, `SMTP_SECURITY`, `SMTP_TIMEOUT`, `SMTP_FROM`,
`SMTP_FROM_NAME`, and `APP_FRONTEND_URL`. The tracked production example uses
`SMTP_SECURITY=starttls` on port 587. `tls` supports implicit TLS (commonly
port 465). `none` is for local MailHog only and Auth rejects it in release
mode. TLS certificates are verified normally; do not disable verification.

Use a verified sender identity such as `AstraCode <no-reply@your-domain>`.
The sending domain/provider must be configured with SPF, DKIM, and DMARC using
the DNS records supplied by the selected provider. Those records are provider
and domain-specific and are not application configuration.

`APP_FRONTEND_URL` must be the real HTTPS public origin, without credentials,
query, or fragment. Auth constructs `/verify-email#token=...` and
`/reset-password?token=...` from that single origin. Ensure the TLS ingress and
frontend domain are live before enabling email delivery; password-reset links
must not be sent over plaintext HTTP in production.

## One-time VPS bootstrap

### 1. Install runtime prerequisites

Install on a supported Linux host:

- Docker Engine;
- Docker Compose plugin;
- `curl`;
- OpenSSH server for deployment automation;
- a host TLS reverse proxy such as Caddy, Nginx, or an equivalent managed edge.

The v0.1.0 release workflow publishes `linux/amd64` images. Use an x86-64 VPS;
multi-architecture publishing is intentionally not claimed by this release.

Verify:

```bash
docker version
docker compose version
curl --version
```

Create stable directories and make the deployment user the owner of the
non-secret application directory:

```bash
sudo install -d -m 0755 -o DEPLOY_USER -g DEPLOY_USER /opt/astracode
sudo groupadd --system astracode-secrets
sudo usermod -aG astracode-secrets DEPLOY_USER
sudo install -d -m 0750 -o root -g astracode-secrets /etc/astracode
```

Replace `DEPLOY_USER` with the dedicated SSH deployment account. That account
needs permission to use Docker and read the `astracode-secrets` group. Start a
new login session after adding group membership. Do not grant it broader
privileges than the deployment requires.

### 2. Provision production environment files

From a trusted checkout/workstation, use these tracked templates:

```text
environment/postgres.prod.env.example -> /etc/astracode/postgres.env
environment/redis.prod.env.example    -> /etc/astracode/redis.env
environment/service.prod.env.example  -> /etc/astracode/service.env
environment/stack.env.example         -> /etc/astracode/stack.env
```

Copy them through your secret-provisioning channel, replace every `REPLACE_*`
value, and restrict access:

```bash
sudo chown root:astracode-secrets /etc/astracode/*.env
sudo chmod 0640 /etc/astracode/*.env
```

Important relationships:

- `POSTGRES_PASSWORD` and `DATABASE_PASSWORD` must match across the PostgreSQL
  and service files.
- `REDIS_PASSWORD` must match across the Redis and service files.
- `MINIO_ROOT_USER`/`MINIO_ROOT_PASSWORD` must match the service-side
  `MINIO_ACCESS_KEY`/`MINIO_SECRET_KEY`.
- `JWT_ACCESS_SECRET` must match the symmetric JWK mounted into KrakenD.
- `APP_FRONTEND_URL` and `SSE_ALLOWED_ORIGIN` must use the real public HTTPS
  origin.
- Configure the SMTP variables in `service.env` with the transactional
  provider's authenticated STARTTLS or TLS settings. Do not use MailHog,
  localhost, or a plaintext SMTP connection in production.
- Keep `MINIO_PUBLIC_URL=/` for same-origin avatar paths in this topology.
- `ASTRACODE_IMAGE_ROOT` must be
  `ghcr.io/<owner>/<repository>`; the release workflow appends component names.
- Keep `ASTRACODE_EDGE_BIND=127.0.0.1` when TLS terminates on the same host.

Generate independent high-entropy values. Do not reuse development examples.
Avoid putting generated secrets in shell history, chat, CI output, or Git.

### 3. Generate the production JWK

The repository's `gateway/gen_jwk.go` converts the raw HS256 access secret into
the KrakenD JWK representation and writes it with mode `0600`:

```bash
umask 077
export JWT_ACCESS_SECRET='THE_SAME_RANDOM_VALUE_USED_IN_SERVICE_ENV'
JWK_OUTPUT_PATH=/tmp/astracode-symmetric.json go run ./gateway/gen_jwk.go
unset JWT_ACCESS_SECRET
sudo install -m 0640 -o root -g astracode-secrets \
  /tmp/astracode-symmetric.json /etc/astracode/symmetric.json
```

Run this on a trusted workstation if Go is intentionally absent from the VPS,
then transfer the file through the same secure bootstrap channel. Never use the
tracked local-development JWK in production.

### 4. Authenticate the VPS to GHCR

Public GHCR packages need no registry login. If the packages are private, log
in once on the VPS using a least-privilege read token supplied interactively or
by your secret manager:

```bash
docker login ghcr.io -u GITHUB_USERNAME --password-stdin
```

Do not store this token in the repository or production Compose files.

### 5. Configure TLS and firewall rules

Terminate TLS at the host/managed ingress and proxy the public origin to
`http://127.0.0.1:8080`. Preserve the `Host` and standard forwarded headers.
For `/events/`, disable response buffering and allow long-lived streaming
connections.

The Compose configuration deliberately does not invent a domain or acquire a
certificate. The operator must provide the real hostname, DNS, TLS certificate,
and renewal policy.

Internet-facing firewall rules should normally expose only:

- TCP 80/443 for the TLS ingress;
- SSH according to operator policy.

Do not expose PostgreSQL, Redis, Kafka, MinIO API/console, KrakenD, service HTTP
or gRPC ports, Judge Worker, or the executor.

### 6. Configure GitHub production deployment

Create a GitHub Environment named `production`. Add required reviewers and
deployment-branch restrictions as appropriate. Configure these secrets by name:

| Secret | Purpose |
| --- | --- |
| `PROD_HOST` | VPS host or IP. |
| `PROD_PORT` | SSH port. |
| `PROD_USER` | Dedicated deployment account. |
| `PROD_SSH_KEY` | Private SSH key for that account. |
| `PROD_KNOWN_HOSTS` | Pinned `known_hosts` entry for strict host verification. |

No database, JWT, Redis, MinIO, SMTP, or SSE secret belongs in this workflow.
The deploy account must be able to create files under `/opt/astracode`, use
Docker, and read the externally provisioned environment files through Compose.

## Publishing a release

After CI and release-candidate validation, creating and pushing a semantic tag
such as `v0.1.0` triggers the image workflow. It publishes:

```text
ghcr.io/<owner>/<repository>/website:v0.1.0
ghcr.io/<owner>/<repository>/auth-service:v0.1.0
ghcr.io/<owner>/<repository>/problem-service:v0.1.0
ghcr.io/<owner>/<repository>/submission-service:v0.1.0
ghcr.io/<owner>/<repository>/judge-worker:v0.1.0
ghcr.io/<owner>/<repository>/sandbox:v0.1.0
```

Each image also receives `sha-<full-commit>`. No production path relies on
`latest`. Artifact publication does not deploy automatically.

## Normal deployment

Preferred: run the **Deploy production** workflow, enter a published `vX.Y.Z`
or full `sha-<commit>` tag, and approve the `production` Environment gate. The
workflow uploads the current non-secret Compose/config bundle and runs:

```bash
/opt/astracode/scripts/deploy-production.sh v0.1.0
```

The script:

1. validates the immutable tag;
2. loads `/etc/astracode/stack.env`;
3. runs `docker compose pull`;
4. runs `docker compose up -d --remove-orphans` without `down`;
5. performs bounded Envoy and website smoke checks;
6. keeps the avatar bucket publicly downloadable while testcase storage stays private;
7. records the current and previous successful image tags.

It never prunes Docker state or deletes volumes. Compose may recreate changed
containers, so brief interruptions remain possible; this is not zero-downtime
orchestration.

The equivalent direct operator command is:

```bash
ASTRACODE_STACK_ENV=/etc/astracode/stack.env \
  /opt/astracode/scripts/deploy-production.sh v0.1.0
```

## Rollback

Redeploy any previously published immutable version through the manual workflow,
or use the previous successful tag recorded by the deployment script:

```bash
/opt/astracode/scripts/deploy-production.sh --rollback
```

On a failed activation or smoke test, the script attempts to reactivate the
previous successful application image tag. It does not delete or recreate
persistent data.

Application rollback is not database rollback. Auth, Problem, and Submission
currently use GORM `AutoMigrate`; a release that changes schema may not be
backward compatible with an older application image. Review schema changes and
take a verified database backup before such a release.

## Operations and security checklist

Before public use:

- Back up PostgreSQL and MinIO, document retention, and test restoration.
- Monitor free disk, container health/restarts, Kafka lag, database capacity,
  and executor resource consumption.
- Retain Docker's bounded log rotation and integrate logs with the operator's
  chosen collection/retention system.
- Keep Docker, the host kernel, Envoy, KrakenD, data stores, and custom images
  patched through intentional release testing.
- Review the privileged `go-judge` container, tmpfs mounts, compiler/runtime
  toolchain, worker concurrency, CPU/memory limits, and host blast radius.
- Keep direct container/service ports unreachable from untrusted networks.
- Rotate credentials on a documented schedule and immediately after suspected
  disclosure.
- [ ] Create the SMTP provider account and verify its sending domain.
- [ ] Configure SPF, DKIM, and DMARC from the provider's DNS instructions.
- [ ] Install SMTP credentials only in `/etc/astracode/service.env`.
- [ ] Set a verified `SMTP_FROM` and `SMTP_FROM_NAME`.
- [ ] Set `APP_FRONTEND_URL` to the HTTPS public frontend origin.
- [ ] Test registration and confirm its verification email/link works.
- [ ] Test forgot-password and confirm its reset email/link works.
- [ ] Confirm MailHog is absent from the production Compose runtime.

The symmetric JWK was historically tracked in this repository. Replacing the
working-tree value does not remove it from Git history. Treat every historical
access secret represented there as compromised and rotate the production JWT
access/refresh secrets before launch. The ignored local `environment/*.env`
files were not found in tracked history during this audit, but any values ever
shared elsewhere still require independent rotation.

## Manual Compose reference

For inspection or controlled operation on the VPS:

```bash
export ASTRACODE_IMAGE_TAG=v0.1.0

docker compose \
  --env-file /etc/astracode/stack.env \
  -f /opt/astracode/docker-compose.yml \
  -f /opt/astracode/docker-compose.prod.yml \
  pull

docker compose \
  --env-file /etc/astracode/stack.env \
  -f /opt/astracode/docker-compose.yml \
  -f /opt/astracode/docker-compose.prod.yml \
  up -d --remove-orphans
```

Never use `docker compose down -v`, volume pruning, or filesystem deletion as a
normal deployment or rollback step.
