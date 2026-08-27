#!/usr/bin/env bash
# Safe launcher for the GET-only k6 public-problem capacity workload.
set -euo pipefail

usage() { cat <<'EOF'
Usage: run.sh --base-url URL --rate RPS --duration Ns|Nm|Nh --preallocated-vus N --max-vus N --max-requests N [--path /api/v1/problems/SLUG] [--allow-remote --confirm-target-host HOST] [--system-config FILE] [--output DIR] [--run-id ID] [--repetition N]
EOF
}

base_url= path=/api/v1/problems/two-sum rate= duration= preallocated= max_vus= max_requests= system_config= output= run_id= repetition=1 allow_remote=false confirm_host=
while (($#)); do
  case "$1" in
    --base-url|--path|--rate|--duration|--preallocated-vus|--max-vus|--max-requests|--system-config|--output|--run-id|--repetition|--confirm-target-host) key=$1; shift; (($#)) || { usage >&2; exit 2; }; value=$1; shift; case "$key" in --base-url) base_url=$value;; --path) path=$value;; --rate) rate=$value;; --duration) duration=$value;; --preallocated-vus) preallocated=$value;; --max-vus) max_vus=$value;; --max-requests) max_requests=$value;; --system-config) system_config=$value;; --output) output=$value;; --run-id) run_id=$value;; --repetition) repetition=$value;; --confirm-target-host) confirm_host=$value;; esac ;;
    --allow-remote) allow_remote=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "api-bench: unknown argument $1" >&2; usage >&2; exit 2 ;;
  esac
done
command -v k6 >/dev/null || { echo 'api-bench: k6 is required; install k6 separately' >&2; exit 127; }
[[ $base_url && $rate && $duration && $preallocated && $max_vus && $max_requests ]] || { echo 'api-bench: required arguments missing' >&2; exit 2; }
[[ $path =~ ^/api/v1/problems/[A-Za-z0-9._-]+$ ]] || { echo 'api-bench: --path must select a public problem endpoint' >&2; exit 2; }
[[ $duration =~ ^[1-9][0-9]*[smh]$ ]] || { echo 'api-bench: duration must be a bounded whole Ns, Nm, or Nh value' >&2; exit 2; }
[[ $preallocated =~ ^[1-9][0-9]*$ && $max_vus =~ ^[1-9][0-9]*$ && $max_requests =~ ^[1-9][0-9]*$ && $repetition =~ ^[1-9][0-9]*$ ]] || { echo 'api-bench: VUs, request cap, and repetition must be positive integers' >&2; exit 2; }
awk -v r="$rate" 'BEGIN { exit !(r > 0) }' || { echo 'api-bench: rate must be positive' >&2; exit 2; }
(( max_vus >= preallocated )) || { echo 'api-bench: max VUs must be at least preallocated VUs' >&2; exit 2; }
seconds=${duration%[smh]}; unit=${duration: -1}; case "$unit" in m) seconds=$((seconds*60));; h) seconds=$((seconds*3600));; esac
expected=$(python3 - "$rate" "$seconds" <<'PY'
from decimal import Decimal, ROUND_CEILING
import sys
try:
    value = Decimal(sys.argv[1]) * Decimal(sys.argv[2])
    if value <= 0:
        raise ValueError
    print(value.to_integral_value(rounding=ROUND_CEILING))
except Exception:
    raise SystemExit(2)
PY
) || { echo 'api-bench: rate must be a positive decimal' >&2; exit 2; }
(( expected <= max_requests )) || { echo "api-bench: expected request budget $expected exceeds --max-requests $max_requests" >&2; exit 2; }
parsed=$(python3 - "$base_url" <<'PY'
from urllib.parse import urlparse
import sys
u=urlparse(sys.argv[1]); print(f"{u.scheme}|{u.hostname or ''}|{u.path}|{u.query}|{u.fragment}|{u.username or ''}")
PY
)
IFS='|' read -r scheme host url_path query fragment username <<<"$parsed"
[[ $host && -z $query && -z $fragment && -z $username && ( -z $url_path || $url_path == / ) ]] || { echo 'api-bench: invalid base URL' >&2; exit 2; }
loopback=false; [[ $host == localhost || $host == 127.0.0.1 || $host == ::1 ]] && loopback=true
if [[ $loopback == false ]]; then
  [[ $scheme == https && $allow_remote == true && $confirm_host == "$host" ]] || { echo 'api-bench: remote targets require HTTPS, --allow-remote, and exact --confirm-target-host' >&2; exit 2; }
fi
[[ -z $system_config || -f $system_config ]] || { echo 'api-bench: --system-config must be a regular file' >&2; exit 2; }
if [[ -n $system_config ]]; then
  system_config=$(cd "$(dirname "$system_config")" && pwd)/$(basename "$system_config")
fi
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
run_id=${run_id:-API-R${repetition}-$(date -u +%Y%m%dT%H%M%SZ)}
output=${output:-"$root/tools/benchmark/api/bench-results/$run_id"}
[[ ! -e $output ]] || { echo 'api-bench: output directory already exists' >&2; exit 2; }
mkdir -p -m 700 "$output"
git_sha=$(git -C "$root" rev-parse HEAD); git_dirty=false; git -C "$root" diff --quiet || git_dirty=true
export BASE_URL="${base_url%/}" API_PATH="$path" TARGET_RPS="$rate" DURATION="$duration" PREALLOCATED_VUS="$preallocated" MAX_VUS="$max_vus" MAX_REQUESTS="$max_requests" SYSTEM_CONFIG_PATH="$system_config" OUTPUT_DIR="$output" RUN_ID="$run_id" STARTED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)" GIT_SHA="$git_sha" GIT_DIRTY="$git_dirty"
exec k6 run "$(dirname "${BASH_SOURCE[0]}")/api.js"
