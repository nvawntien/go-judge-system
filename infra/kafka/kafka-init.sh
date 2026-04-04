#!/usr/bin/env bash
set -euo pipefail

KAFKA_BOOTSTRAP_SERVER="${KAFKA_BOOTSTRAP_SERVER:-kafka:9092}"

/opt/bitnami/kafka/bin/kafka-topics.sh --bootstrap-server "$KAFKA_BOOTSTRAP_SERVER" --create --if-not-exists --topic judge.submission.jobs --partitions 3 --replication-factor 1
/opt/bitnami/kafka/bin/kafka-topics.sh --bootstrap-server "$KAFKA_BOOTSTRAP_SERVER" --create --if-not-exists --topic judge.submission.jobs.dlt --partitions 1 --replication-factor 1
/opt/bitnami/kafka/bin/kafka-topics.sh --bootstrap-server "$KAFKA_BOOTSTRAP_SERVER" --create --if-not-exists --topic judge.submission.results --partitions 3 --replication-factor 1
/opt/bitnami/kafka/bin/kafka-topics.sh --bootstrap-server "$KAFKA_BOOTSTRAP_SERVER" --list
