#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'EOF'
Usage: collect-kafka-lag.sh --bootstrap-server HOST:PORT --consumer-group GROUP --topic TOPIC [options]

Collects a low-frequency Kafka consumer-group snapshot as normalized CSV. It is
intended for an operator-run auxiliary monitor, not for the benchmark client.

Options:
  --bootstrap-server ADDRESS  Kafka bootstrap address (required)
  --consumer-group GROUP     Consumer group to inspect (required)
  --topic TOPIC              Topic to inspect (required)
  --command PATH             kafka-consumer-groups.sh; default: kafka-consumer-groups.sh
  --interval DURATION        Seconds between samples; default: 15
  --raw-output PATH          Append timestamped raw CLI output to PATH
  --once                     Emit one sample and exit
  -h, --help                 Show this help
EOF
}

bootstrap=""
group=""
topic=""
kafka_command=kafka-consumer-groups.sh
interval=15
raw_output=""
once=0

while (($# > 0)); do
  case "$1" in
    --bootstrap-server) bootstrap=${2:?missing value for --bootstrap-server}; shift 2 ;;
    --consumer-group) group=${2:?missing value for --consumer-group}; shift 2 ;;
    --topic) topic=${2:?missing value for --topic}; shift 2 ;;
    --command) kafka_command=${2:?missing value for --command}; shift 2 ;;
    --interval) interval=${2:?missing value for --interval}; shift 2 ;;
    --raw-output) raw_output=${2:?missing value for --raw-output}; shift 2 ;;
    --once) once=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

[[ -n "$bootstrap" && -n "$group" && -n "$topic" ]] || {
  echo "--bootstrap-server, --consumer-group, and --topic are required" >&2
  exit 2
}
[[ "$interval" =~ ^([0-9]+([.][0-9]+)?|[.][0-9]+)$ ]] || {
  echo "--interval must be a positive number of seconds" >&2
  exit 2
}
awk -v value="$interval" 'BEGIN { exit !(value + 0 > 0) }' || {
  echo "--interval must be greater than zero" >&2
  exit 2
}
command -v "$kafka_command" >/dev/null 2>&1 || {
  echo "Kafka CLI command not found: $kafka_command" >&2
  exit 1
}

sample() {
  local timestamp raw
  timestamp=$(date -u +%Y-%m-%dT%H:%M:%S.%NZ)
  if ! raw=$("$kafka_command" --bootstrap-server "$bootstrap" --describe --group "$group" --topic "$topic" 2>&1); then
    echo "Kafka consumer-group query failed at $timestamp" >&2
    [[ -n "$raw_output" ]] && {
      printf '# %s query_failed\n%s\n' "$timestamp" "$raw" >>"$raw_output"
    }
    return 1
  fi

  [[ -n "$raw_output" ]] && {
    printf '# %s\n%s\n' "$timestamp" "$raw" >>"$raw_output"
  }

  awk -v timestamp="$timestamp" -v group="$group" -v topic="$topic" '
    $1 == group && $2 == topic && $3 ~ /^[0-9]+$/ &&
      $4 ~ /^-?[0-9]+$/ && $5 ~ /^-?[0-9]+$/ && $6 ~ /^-?[0-9]+$/ {
        printf "%s,%s,%s,%s,%s,%s,%s\\n", timestamp, group, topic, $3, $4, $5, $6
      }
  ' <<<"$raw"
}

printf 'timestamp,consumer_group,topic,partition,current_offset,log_end_offset,lag\n'
while :; do
  sample
  ((once == 1)) && break
  sleep "$interval"
done
