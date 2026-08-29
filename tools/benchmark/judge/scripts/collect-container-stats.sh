#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'EOF'
Usage: collect-container-stats.sh --node NAME --container NAME [options]

Collects local Docker container samples as CSV. Repeat --container for each
explicitly selected container; the script never guesses production containers.

Options:
  --node NAME          Logical node label written to CSV (required)
  --container NAME     Docker container to sample (repeatable, required)
  --interval DURATION  Seconds between samples; default: 2
  --once               Emit one sample and exit
  -h, --help           Show this help
EOF
}

node=""
interval=2
once=0
declare -a containers=()

while (($# > 0)); do
  case "$1" in
    --node)
      node=${2:?missing value for --node}
      shift 2
      ;;
    --container)
      containers+=("${2:?missing value for --container}")
      shift 2
      ;;
    --interval)
      interval=${2:?missing value for --interval}
      shift 2
      ;;
    --once)
      once=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

[[ -n "$node" ]] || { echo "--node is required" >&2; exit 2; }
((${#containers[@]} > 0)) || { echo "at least one --container is required" >&2; exit 2; }
[[ "$interval" =~ ^([0-9]+([.][0-9]+)?|[.][0-9]+)$ ]] || {
  echo "--interval must be a positive number of seconds" >&2
  exit 2
}

awk -v value="$interval" 'BEGIN { exit !(value + 0 > 0) }' || {
  echo "--interval must be greater than zero" >&2
  exit 2
}

command -v docker >/dev/null 2>&1 || { echo "docker is required" >&2; exit 1; }

to_bytes() {
  awk -v raw="$1" '
    function factor(unit) {
      if (unit == "B") return 1
      if (unit == "kB" || unit == "KB") return 1000
      if (unit == "MB") return 1000000
      if (unit == "GB") return 1000000000
      if (unit == "KiB") return 1024
      if (unit == "MiB") return 1048576
      if (unit == "GiB") return 1073741824
      if (unit == "TiB") return 1099511627776
      return 0
    }
    BEGIN {
      if (raw !~ /^[0-9.]+[A-Za-z]+$/) exit 1
      value = raw
      unit = raw
      sub(/^[0-9.]+/, "", unit)
      sub(/[A-Za-z]+$/, "", value)
      f = factor(unit)
      if (f == 0) exit 1
      printf "%.0f", value * f
    }
  '
}

sample_container() {
  local timestamp container stat cpu mem_usage mem_limit mem_percent pids net_io block_io usage_bytes limit_bytes inspect restart_count state health
  timestamp=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
  container=$1

  if ! stat=$(docker stats --no-stream --format '{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.PIDs}}\t{{.NetIO}}\t{{.BlockIO}}' "$container" 2>/dev/null); then
    echo "unable to sample container: $container" >&2
    return 0
  fi

  IFS=$'\t' read -r cpu mem_usage mem_percent pids net_io block_io <<<"$stat"
  if ! inspect=$(docker inspect --format '{{.RestartCount}}\t{{.State.Status}}\t{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$container" 2>/dev/null); then
    restart_count=unknown; state=unknown; health=unknown
  else
    IFS=$'\t' read -r restart_count state health <<<"$inspect"
  fi
  IFS='/' read -r mem_usage mem_limit <<<"$mem_usage"
  mem_usage=${mem_usage//[[:space:]]/}
  mem_limit=${mem_limit//[[:space:]]/}
  usage_bytes=$(to_bytes "$mem_usage") || { echo "unrecognized Docker memory value: $mem_usage" >&2; return 1; }
  limit_bytes=$(to_bytes "$mem_limit") || { echo "unrecognized Docker memory limit: $mem_limit" >&2; return 1; }

  printf '%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n' \
    "$timestamp" "$node" "$container" \
    "${cpu%%%}" "$usage_bytes" "$limit_bytes" "${mem_percent%%%}" "$pids" "$net_io" "$block_io" "$restart_count" "$state" "$health"
}

printf 'timestamp,node,container,cpu_percent,memory_bytes,memory_limit_bytes,memory_percent,pids,network_io,block_io,restart_count,state,health\n'
while :; do
  for container in "${containers[@]}"; do
    sample_container "$container"
  done
  ((once == 1)) && break
  sleep "$interval"
done
