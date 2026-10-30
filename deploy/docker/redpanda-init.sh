#!/bin/bash
set -e
BROKERS="${KAFKA_BROKERS:-redpanda:9092}"
rpk topic create click_events -p 3 -r 1 --brokers "$BROKERS" 2>/dev/null || true
echo "Redpanda topics ready."
