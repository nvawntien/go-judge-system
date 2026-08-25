#!/usr/bin/env bash
set -euo pipefail

readonly KAFKA_BOOTSTRAP_SERVER="${KAFKA_BOOTSTRAP_SERVER:-kafka:9092}"
readonly KAFKA_TOPICS_COMMAND="/opt/bitnami/kafka/bin/kafka-topics.sh"
readonly KAFKA_READY_MAX_ATTEMPTS=30
readonly KAFKA_READY_RETRY_SECONDS=2

wait_for_kafka_protocol() {
  local attempt

  for ((attempt = 1; attempt <= KAFKA_READY_MAX_ATTEMPTS; attempt++)); do
    if "$KAFKA_TOPICS_COMMAND" --bootstrap-server "$KAFKA_BOOTSTRAP_SERVER" --list >/dev/null 2>&1; then
      echo "Kafka protocol is ready at $KAFKA_BOOTSTRAP_SERVER (attempt $attempt/$KAFKA_READY_MAX_ATTEMPTS)."
      return 0
    fi

    if ((attempt == KAFKA_READY_MAX_ATTEMPTS)); then
      echo "Kafka did not become protocol-ready at $KAFKA_BOOTSTRAP_SERVER after $KAFKA_READY_MAX_ATTEMPTS attempts." >&2
      return 1
    fi

    echo "Kafka protocol is not ready at $KAFKA_BOOTSTRAP_SERVER (attempt $attempt/$KAFKA_READY_MAX_ATTEMPTS); retrying in ${KAFKA_READY_RETRY_SECONDS}s." >&2
    sleep "$KAFKA_READY_RETRY_SECONDS" || return 1
  done

  return 1
}

wait_for_kafka_protocol || exit 1

"$KAFKA_TOPICS_COMMAND" --bootstrap-server "$KAFKA_BOOTSTRAP_SERVER" --create --if-not-exists --topic judge.submission.jobs --partitions 3 --replication-factor 1
"$KAFKA_TOPICS_COMMAND" --bootstrap-server "$KAFKA_BOOTSTRAP_SERVER" --create --if-not-exists --topic judge.submission.jobs.dlt --partitions 1 --replication-factor 1
"$KAFKA_TOPICS_COMMAND" --bootstrap-server "$KAFKA_BOOTSTRAP_SERVER" --create --if-not-exists --topic judge.submission.results --partitions 3 --replication-factor 1
"$KAFKA_TOPICS_COMMAND" --bootstrap-server "$KAFKA_BOOTSTRAP_SERVER" --list
