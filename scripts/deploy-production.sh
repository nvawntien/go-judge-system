#!/usr/bin/env bash
set -Eeuo pipefail

readonly APP_DIR="/opt/astracode"
readonly STATE_DIR="$APP_DIR/.deploy"
readonly STACK_ENV="${ASTRACODE_STACK_ENV:-/etc/astracode/stack.env}"
readonly CURRENT_TAG_FILE="$STATE_DIR/current-image-tag"
readonly PREVIOUS_TAG_FILE="$STATE_DIR/previous-image-tag"
readonly COMPOSE_FILES=(-f "$APP_DIR/docker-compose.yml" -f "$APP_DIR/docker-compose.prod.yml")

usage() {
  echo "usage: $0 <vX.Y.Z|sha-COMMIT> | --rollback" >&2
}

valid_tag() {
  [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ || "$1" =~ ^sha-[0-9a-f]{40}$ ]]
}

if [[ ! -r "$STACK_ENV" ]]; then
  echo "Missing production stack environment: $STACK_ENV" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$STACK_ENV"
set +a

requested="${1:-}"
if [[ "$requested" == "--rollback" ]]; then
  if [[ ! -r "$PREVIOUS_TAG_FILE" ]]; then
    echo "No previous successful image tag is recorded." >&2
    exit 1
  fi
  requested="$(<"$PREVIOUS_TAG_FILE")"
elif [[ $# -ne 1 ]]; then
  usage
  exit 2
fi

if ! valid_tag "$requested"; then
  echo "Refusing invalid or mutable image tag: $requested" >&2
  exit 2
fi

if [[ -z "${ASTRACODE_IMAGE_ROOT:-}" ]]; then
  echo "ASTRACODE_IMAGE_ROOT is required in $STACK_ENV" >&2
  exit 1
fi

export ASTRACODE_IMAGE_TAG="$requested"
mkdir -p "$STATE_DIR"

previous=""
if [[ -r "$CURRENT_TAG_FILE" ]]; then
  previous="$(<"$CURRENT_TAG_FILE")"
fi

compose() {
  docker compose --env-file "$STACK_ENV" "${COMPOSE_FILES[@]}" "$@"
}

smoke_test() {
  local edge_bind="${ASTRACODE_EDGE_BIND:-127.0.0.1}"
  local edge_port="${ASTRACODE_EDGE_PORT:-8080}"
  local smoke_host="$edge_bind"

  if [[ "$smoke_host" == "0.0.0.0" || "$smoke_host" == "::" ]]; then
    smoke_host="127.0.0.1"
  fi

  for _ in {1..24}; do
    if curl --fail --silent --show-error --max-time 5 "http://$smoke_host:$edge_port/envoy-health" >/dev/null && \
       curl --fail --silent --show-error --max-time 10 "http://$smoke_host:$edge_port/" >/dev/null; then
      return 0
    fi
    sleep 5
  done

  return 1
}

verify_services() {
  local service container_id running health
  local services=(
    website envoy gateway postgres redis minio kafka go-judge
    auth-service problem-service submission-service judge-worker
  )

  for service in "${services[@]}"; do
    container_id="$(compose ps -q "$service")"
    if [[ -z "$container_id" ]]; then
      echo "Required service has no container: $service" >&2
      return 1
    fi

    running="$(docker inspect --format '{{.State.Running}}' "$container_id")"
    if [[ "$running" != "true" ]]; then
      echo "Required service is not running: $service" >&2
      return 1
    fi

    health="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "$container_id")"
    if [[ -n "$health" && "$health" != "healthy" ]]; then
      echo "Required service is not healthy: $service ($health)" >&2
      return 1
    fi
  done
}

activate() {
  compose pull
  compose up -d --remove-orphans
  compose ps
  smoke_test
  verify_services

  # Auth creates the avatars bucket. This preserves the product's existing
  # public-avatar contract while hidden testcase buckets remain private.
  compose exec -T minio sh -ec \
    'mc alias set local http://127.0.0.1:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null && mc anonymous set download local/avatars >/dev/null'
}

echo "Deploying AstraCode image tag $ASTRACODE_IMAGE_TAG"
if ! activate; then
  echo "Deployment or smoke test failed for $ASTRACODE_IMAGE_TAG." >&2
  compose ps || true

  if [[ -n "$previous" && "$previous" != "$ASTRACODE_IMAGE_TAG" ]] && valid_tag "$previous"; then
    echo "Attempting automatic rollback to $previous" >&2
    export ASTRACODE_IMAGE_TAG="$previous"
    activate || echo "Rollback activation also failed; operator action is required." >&2
  else
    echo "No distinct previous successful release is available for automatic rollback." >&2
  fi
  exit 1
fi

if [[ -n "$previous" && "$previous" != "$requested" ]]; then
  printf '%s\n' "$previous" > "$PREVIOUS_TAG_FILE.tmp"
  mv "$PREVIOUS_TAG_FILE.tmp" "$PREVIOUS_TAG_FILE"
fi
printf '%s\n' "$requested" > "$CURRENT_TAG_FILE.tmp"
mv "$CURRENT_TAG_FILE.tmp" "$CURRENT_TAG_FILE"

echo "AstraCode $requested is healthy."
