#!/usr/bin/env bash
set -Eeuo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

run_case() {
  local name="$1"
  local attempts="$2"
  local succeeds_on="$3"
  local expected="$4"
  local calls=0

  set +e
  (
    export ASTRACODE_JUDGE_DEPLOY_TEST_LIB=1
    export ASTRACODE_SANDBOX_GRPC_READY_ATTEMPTS="$attempts"
    export ASTRACODE_SANDBOX_GRPC_READY_INTERVAL_SECONDS=0
    local_dir="$(mktemp -d)"
    : >"$local_dir/judge.compose.yml"
    printf 'grpc_port: 9094\n' >"$local_dir/config.yaml"
    export ASTRACODE_JUDGE_COMPOSE="$local_dir/judge.compose.yml"
    export ASTRACODE_JUDGE_CONFIG="$local_dir/config.yaml"
    export ASTRACODE_JUDGE_STATE_DIR="$local_dir/state"
    set -- v1.0.0
    # shellcheck source=deploy-judge-node.sh
    source "$script_dir/deploy-judge-node.sh"
    verify_listening_tcp_port() {
      calls=$((calls + 1))
      [[ "$calls" -ge "$succeeds_on" ]]
    }
    wait_for_listening_tcp_port sandbox 5051
  )
  local status=$?
  set -e
  if [[ "$expected" == pass && "$status" -ne 0 ]] || [[ "$expected" == fail && "$status" -eq 0 ]]; then
    echo "readiness case failed: $name status=$status" >&2
    return 1
  fi
}

run_case "grpc already listening" 3 1 pass
run_case "http-only/grpc closed" 2 99 fail
run_case "delayed grpc listener" 3 2 pass
run_case "grpc never appears" 2 99 fail
