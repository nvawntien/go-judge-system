#!/usr/bin/env bash
set -Eeuo pipefail

# Judge Compose and config are node-owned operational files. Deployment only
# creates a temporary image override; it never rewrites either external file.
readonly JUDGE_COMPOSE="${ASTRACODE_JUDGE_COMPOSE:-/etc/astracode/judge-node.compose.yml}"
readonly JUDGE_CONFIG="${ASTRACODE_JUDGE_CONFIG:-/etc/astracode/judge/config.yaml}"
readonly IMAGE_ROOT="${ASTRACODE_JUDGE_IMAGE_ROOT:-ghcr.io/nvawntien/go-judge-system}"
readonly STATE_DIR="${ASTRACODE_JUDGE_STATE_DIR:-/opt/astracode/.deploy/judge}"
readonly CURRENT_TAG_FILE="$STATE_DIR/current-image-tag"
readonly PREVIOUS_TAG_FILE="$STATE_DIR/previous-image-tag"
readonly JUDGE_SERVICES=(go-judge judge-worker)

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

  for container in judge_worker judge_sandbox; do
    if ! image="$(docker inspect --format '{{.Config.Image}}' "$container" 2>/dev/null)"; then
      echo "Cannot inspect running Judge release container: $container" >&2
      return 1
    fi

    if ! tag="$(release_tag_from_image "$image")"; then
      echo "Container $container does not use a valid semantic release image: $image" >&2
      return 1
    fi

    if [[ -n "$runtime_tag" && "$runtime_tag" != "$tag" ]]; then
      echo "Running Judge release containers disagree on their image tag: $runtime_tag != $tag" >&2
      return 1
    fi
    runtime_tag="$tag"
  done

  if [[ -z "$runtime_tag" ]]; then
    echo "Cannot determine the currently active Judge release." >&2
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
      echo "Deployment state drift: Judge state=$state_tag runtime=$runtime_tag" >&2
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

require_readable_file "$JUDGE_COMPOSE" "Judge Compose file"
require_readable_file "$JUDGE_CONFIG" "Judge configuration"

requested="${1:-}"
if [[ "$requested" == "--rollback" ]]; then
  require_readable_file "$PREVIOUS_TAG_FILE" "previous Judge release state"
  requested="$(<"$PREVIOUS_TAG_FILE")"
elif [[ $# -ne 1 ]]; then
  usage
  exit 2
fi

if ! valid_tag "$requested"; then
  echo "Refusing non-release or mutable image tag: $requested" >&2
  exit 2
fi

mkdir -p "$STATE_DIR"

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/astracode-judge-deploy.XXXXXX")"
readonly tmp_dir
readonly image_override="$tmp_dir/images.yml"
trap 'rm -rf "$tmp_dir"' EXIT

write_image_override() {
  cat >"$image_override" <<EOF
services:
  go-judge:
    image: ${IMAGE_ROOT}/sandbox:${requested}
  judge-worker:
    image: ${IMAGE_ROOT}/judge-worker:${requested}
EOF
  chmod 0600 "$image_override"
}

compose() {
  docker compose \
    -f "$JUDGE_COMPOSE" \
    -f "$image_override" \
    "$@"
}

container_is_running() {
  local service="$1"
  local container_id running health
  container_id="$(compose ps -q "$service")"
  if [[ -z "$container_id" ]]; then
    echo "Required Judge service has no container: $service" >&2
    return 1
  fi

  running="$(docker inspect --format '{{.State.Running}}' "$container_id")"
  if [[ "$running" != "true" ]]; then
    echo "Required Judge service is not running: $service" >&2
    return 1
  fi

  health="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' "$container_id")"
  if [[ -n "$health" && "$health" != "healthy" ]]; then
    echo "Required Judge service is not healthy: $service ($health)" >&2
    return 1
  fi
}

read_grpc_port() {
  local port
  port="$(awk '$1 == "grpc_port:" { print $2; exit }' "$JUDGE_CONFIG")"
  if [[ ! "$port" =~ ^[1-9][0-9]{0,4}$ ]] || ((port > 65535)); then
    echo "Unable to read a valid server.grpc_port from $JUDGE_CONFIG" >&2
    return 1
  fi
  printf '%s\n' "$port"
}

verify_listening_tcp_port() {
  local container_id="$1"
  local port="$2"
  local port_hex
  printf -v port_hex '%04X' "$port"
  docker exec "$container_id" sh -ec \
    "awk -v port='$port_hex' 'NR > 1 { split(\$2, address, \":\"); if (toupper(address[2]) == port && \$4 == \"0A\") found = 1 } END { exit !found }' /proc/net/tcp /proc/net/tcp6"
}

verify_sandbox_http() {
  local sandbox_id
  sandbox_id="$(compose ps -q go-judge)"
  # Executorserver has no repository-defined health path. Accepting either a
  # success or an HTTP error response still proves its local HTTP listener is
  # reachable without publishing port 5050. wget uses status 8 for HTTP 4xx/5xx.
  docker exec "$sandbox_id" sh -ec \
    'wget -S -T 5 -O /dev/null http://127.0.0.1:5050/ >/dev/null 2>&1 || test "$?" -eq 8'
}

verify_sandbox_grpc() {
  local sandbox_id
  sandbox_id="$(compose ps -q go-judge)"
  # executorserver v1.7.1 does not expose grpc_health_v1. A listening private
  # TCP socket is the strongest dependency-free readiness primitive available.
  verify_listening_tcp_port "$sandbox_id" 5051
}

verify_services() {
  local worker_id grpc_port
  container_is_running go-judge
  container_is_running judge-worker
  verify_sandbox_http
  verify_sandbox_grpc

  worker_id="$(compose ps -q judge-worker)"
  grpc_port="$(read_grpc_port)"
  verify_listening_tcp_port "$worker_id" "$grpc_port"
}

activate() {
  compose config --quiet
  compose pull "${JUDGE_SERVICES[@]}"
  # Limit Compose to the two Judge services. Docker Compose recreates them with
  # its normal stop signal/timeout; no force-kill or global cleanup is used.
  compose up -d --force-recreate --timeout "${COMPOSE_STOP_TIMEOUT:-30}" "${JUDGE_SERVICES[@]}"
  compose ps "${JUDGE_SERVICES[@]}"
  verify_services
}

write_state() {
  local path="$1"
  local value="$2"
  printf '%s\n' "$value" >"$path.tmp"
  mv "$path.tmp" "$path"
}

write_image_override

if [[ "${ASTRACODE_JUDGE_DEPLOY_DRY_RUN:-0}" == "1" ]]; then
  cat "$image_override"
  exit 0
fi

if ! previous="$(detect_current_release)"; then
  echo "Refusing to mutate Judge containers without a consistent current release tag." >&2
  exit 1
fi

echo "Deploying Judge Node image tag $requested"
if ! activate; then
  echo "Judge deployment verification failed for $requested." >&2
  compose ps "${JUDGE_SERVICES[@]}" || true

  if [[ -n "$previous" && "$previous" != "$requested" ]] && valid_tag "$previous"; then
    echo "Attempting Judge Node rollback to $previous" >&2
    requested="$previous"
    write_image_override
    activate || echo "Judge rollback activation also failed; operator action is required." >&2
  else
    echo "No distinct previous Judge release is available for automatic rollback." >&2
  fi
  exit 1
fi

if [[ -n "$previous" && "$previous" != "$requested" ]]; then
  write_state "$PREVIOUS_TAG_FILE" "$previous"
fi
write_state "$CURRENT_TAG_FILE" "$requested"

echo "Judge Node $requested is healthy."
