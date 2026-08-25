#!/usr/bin/env bash
set -Eeuo pipefail

# The files under /etc/astracode are owned by the App Node. This script reads
# them, but deployment bundles must never replace them.
readonly APP_DIR="${ASTRACODE_APP_DIR:-/opt/astracode}"
readonly STACK_ENV="${ASTRACODE_STACK_ENV:-/etc/astracode/stack.env}"
readonly APP_OVERRIDE="${ASTRACODE_APP_OVERRIDE:-/etc/astracode/app-node.override.yml}"
readonly APP_COMPOSE_PROJECT="${ASTRACODE_APP_COMPOSE_PROJECT:-go-judge-system}"
readonly STATE_DIR="${ASTRACODE_APP_STATE_DIR:-$APP_DIR/.deploy/app}"
readonly CURRENT_TAG_FILE="$STATE_DIR/current-image-tag"
readonly PREVIOUS_TAG_FILE="$STATE_DIR/previous-image-tag"
readonly HEALTH_CHECK_ATTEMPTS="${ASTRACODE_APP_HEALTH_CHECK_ATTEMPTS:-24}"
readonly HEALTH_CHECK_INTERVAL_SECONDS="${ASTRACODE_APP_HEALTH_CHECK_INTERVAL_SECONDS:-5}"
readonly APP_SERVICES=(
  website envoy gateway postgres redis minio kafka
  auth-service problem-service submission-service
)
readonly APP_RELEASE_SERVICES=(
  website auth-service problem-service submission-service
)
readonly APP_RELEASE_CONTAINERS=(
  astracode_website judge_auth judge_problem judge_submission
)

usage() {
  echo "usage: $0 <vX.Y.Z> | --rollback" >&2
}

valid_tag() {
  [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]
}

release_tag_from_image() {
  local image_ref="${1%@*}"
  local tag

  if [[ "$image_ref" != *:* ]]; then
    return 1
  fi

  tag="${image_ref##*:}"
  valid_tag "$tag" || return 1
  printf '%s\n' "$tag"
}

detect_current_release() {
  local state_tag=""
  local container image tag runtime_tag=""

  for container in "${APP_RELEASE_CONTAINERS[@]}"; do
    if ! image="$(docker inspect --format '{{.Config.Image}}' "$container" 2>/dev/null)"; then
      echo "Cannot inspect running App release container: $container" >&2
      return 1
    fi

    if ! tag="$(release_tag_from_image "$image")"; then
      echo "Container $container does not use a valid semantic release image: $image" >&2
      return 1
    fi

    if [[ -n "$runtime_tag" && "$runtime_tag" != "$tag" ]]; then
      echo "Running App release containers disagree on their image tag: $runtime_tag != $tag" >&2
      return 1
    fi
    runtime_tag="$tag"
  done

  if [[ -z "$runtime_tag" ]]; then
    echo "Cannot determine the currently active App release." >&2
    return 1
  fi

  if [[ -e "$CURRENT_TAG_FILE" ]]; then
    if [[ ! -r "$CURRENT_TAG_FILE" ]]; then
      echo "Deployment state file is unreadable: $CURRENT_TAG_FILE" >&2
      return 1
    fi
    state_tag="$(tr -d '\r\n' <"$CURRENT_TAG_FILE")"
    if ! valid_tag "$state_tag"; then
      echo "Deployment state file does not contain a valid semantic release tag: $CURRENT_TAG_FILE" >&2
      return 1
    fi
    if [[ "$state_tag" != "$runtime_tag" ]]; then
      echo "Deployment state drift: App state=$state_tag runtime=$runtime_tag" >&2
      return 1
    fi
  fi

  printf '%s\n' "$runtime_tag"
}

require_readable_file() {
  local path="$1"
  local description="$2"
  if [[ ! -r "$path" ]]; then
    echo "Missing or unreadable $description: $path" >&2
    exit 1
  fi
}

require_readable_file "$APP_DIR/docker-compose.yml" "App Compose file"
require_readable_file "$APP_DIR/docker-compose.prod.yml" "App production Compose file"
require_readable_file "$STACK_ENV" "App stack environment"
require_readable_file "$APP_OVERRIDE" "App Node override"

set -a
# shellcheck disable=SC1090
source "$STACK_ENV"
set +a

if [[ -z "${ASTRACODE_IMAGE_ROOT:-}" ]]; then
  echo "ASTRACODE_IMAGE_ROOT is required in $STACK_ENV" >&2
  exit 1
fi

if [[ -z "$APP_COMPOSE_PROJECT" ]]; then
  echo "ASTRACODE_APP_COMPOSE_PROJECT must not be empty" >&2
  exit 1
fi

if [[ ! "$HEALTH_CHECK_ATTEMPTS" =~ ^[1-9][0-9]*$ ]] || [[ ! "$HEALTH_CHECK_INTERVAL_SECONDS" =~ ^[1-9][0-9]*$ ]]; then
  echo "App health-check attempts and interval must be positive integers" >&2
  exit 2
fi

requested="${1:-}"
if [[ "$requested" == "--rollback" ]]; then
  require_readable_file "$PREVIOUS_TAG_FILE" "previous App release state"
  requested="$(<"$PREVIOUS_TAG_FILE")"
elif [[ $# -ne 1 ]]; then
  usage
  exit 2
fi

if ! valid_tag "$requested"; then
  echo "Refusing non-release or mutable image tag: $requested" >&2
  exit 2
fi

export ASTRACODE_IMAGE_TAG="$requested"
mkdir -p "$STATE_DIR"

if ! previous="$(detect_current_release)"; then
  echo "Refusing to mutate App containers without a consistent current release tag." >&2
  exit 1
fi

compose() {
  docker compose \
    --project-name "$APP_COMPOSE_PROJECT" \
    --env-file "$STACK_ENV" \
    -f "$APP_DIR/docker-compose.yml" \
    -f "$APP_DIR/docker-compose.prod.yml" \
    -f "$APP_OVERRIDE" \
    "$@"
}

smoke_test() {
  local edge_bind="${ASTRACODE_EDGE_BIND:-127.0.0.1}"
  local edge_port="${ASTRACODE_EDGE_PORT:-8080}"
  local smoke_host="$edge_bind"

  if [[ "$smoke_host" == "0.0.0.0" || "$smoke_host" == "::" ]]; then
    smoke_host="127.0.0.1"
  fi

  local attempt
  for ((attempt = 1; attempt <= HEALTH_CHECK_ATTEMPTS; attempt++)); do
    if curl --fail --silent --show-error --max-time 5 "http://$smoke_host:$edge_port/envoy-health" >/dev/null && \
      curl --fail --silent --show-error --max-time 10 "http://$smoke_host:$edge_port/" >/dev/null; then
      return 0
    fi
    if ((attempt < HEALTH_CHECK_ATTEMPTS)); then
      sleep "$HEALTH_CHECK_INTERVAL_SECONDS" || return 1
    fi
  done

  return 1
}

verify_services() {
  local service container_id running health
  local has_starting_service=0
  for service in "${APP_SERVICES[@]}"; do
    if ! container_id="$(compose ps -q "$service")"; then
      echo "Unable to inspect required App service: $service" >&2
      return 1
    fi
    if [[ -z "$container_id" ]]; then
      echo "Required App service has no container: $service" >&2
      return 1
    fi

    if ! running="$(docker inspect --format '{{.State.Running}}' "$container_id")"; then
      echo "Unable to inspect App service state: $service" >&2
      return 1
    fi
    if [[ "$running" != "true" ]]; then
      echo "Required App service is not running: $service" >&2
      return 1
    fi

    if ! health="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "$container_id")"; then
      echo "Unable to inspect App service health: $service" >&2
      return 1
    fi
    if [[ "$health" == "starting" ]]; then
      echo "Required App service is still starting: $service" >&2
      has_starting_service=1
    elif [[ -n "$health" && "$health" != "healthy" ]]; then
      echo "Required App service is not healthy: $service ($health)" >&2
      return 1
    fi
  done

  if ((has_starting_service)); then
    return 2
  fi

  return 0
}

wait_for_services() {
  local attempt status

  for ((attempt = 1; attempt <= HEALTH_CHECK_ATTEMPTS; attempt++)); do
    status=0
    verify_services || status=$?
    if ((status == 0)); then
      return 0
    fi
    if ((status != 2)); then
      return "$status"
    fi
    if ((attempt == HEALTH_CHECK_ATTEMPTS)); then
      echo "Timed out waiting for required App services to become healthy" >&2
      return 1
    fi

    echo "Waiting for required App services to become healthy ($attempt/$HEALTH_CHECK_ATTEMPTS)" >&2
    sleep "$HEALTH_CHECK_INTERVAL_SECONDS" || return 1
  done

  return 1
}

configure_avatar_bucket() {
  # Preserve the existing public-avatar contract; testcase storage remains
  # private because this command touches only the avatars bucket.
  compose exec -T minio sh -ec \
    'mc alias set local http://127.0.0.1:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null && mc anonymous set download local/avatars >/dev/null' || return 1
  return 0
}

activate() {
  compose config --quiet || return 1
  compose pull "${APP_RELEASE_SERVICES[@]}" || return 1
  # No --profile judge is passed. go-judge and judge-worker stay disabled by
  # the required external App Node override.
  # Images for upstream/stateful services are intentionally not pulled as part
  # of an AstraCode application release.
  compose up -d --pull never --timeout "${COMPOSE_STOP_TIMEOUT:-30}" "${APP_SERVICES[@]}" || return 1
  compose ps || return 1
  smoke_test || return 1
  wait_for_services || return 1
  configure_avatar_bucket || return 1
  return 0
}

write_state() {
  local path="$1"
  local value="$2"
  printf '%s\n' "$value" >"$path.tmp" || return 1
  mv "$path.tmp" "$path" || return 1
  return 0
}

persist_release_state() {
  if [[ -n "$previous" && "$previous" != "$requested" ]]; then
    write_state "$PREVIOUS_TAG_FILE" "$previous" || return 1
  fi
  write_state "$CURRENT_TAG_FILE" "$requested" || return 1
  return 0
}

rollback_to_previous() {
  if [[ -z "$previous" || "$previous" == "$ASTRACODE_IMAGE_TAG" ]] || ! valid_tag "$previous"; then
    echo "No distinct previous App release is available for automatic rollback." >&2
    return 1
  fi

  echo "Attempting App Node rollback to $previous" >&2
  export ASTRACODE_IMAGE_TAG="$previous"
  activate || return 1
  return 0
}

echo "Deploying App Node image tag $ASTRACODE_IMAGE_TAG"
if ! activate; then
  echo "App Node activation failed for $ASTRACODE_IMAGE_TAG." >&2
  compose ps || true
  rollback_to_previous || echo "App rollback activation also failed; operator action is required." >&2
  exit 1
fi

if ! persist_release_state; then
  echo "App release activated but deployment state could not be persisted." >&2
  rollback_to_previous || echo "App rollback activation also failed; operator action is required." >&2
  exit 1
fi

echo "App Node $requested is healthy."
