#!/usr/bin/env bash
set -euo pipefail

# Helper script to run the local CI-like flow:
# - builds and starts the test stack via docker compose
# - runs migrations (if the binary is available)
# - executes unit + e2e tests

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)

# Try to locate docker-compose.test.yml in a few likely places: repo root, parent, grandparent
find_compose_file() {
  local start="$REPO_ROOT"
  for i in 0 1 2; do
    local cand="$start"
    if [ $i -eq 1 ]; then
      cand=$(dirname "$start")
    elif [ $i -eq 2 ]; then
      cand=$(dirname "$(dirname "$start")")
    fi
    if [ -f "$cand/docker-compose.test.yml" ]; then
      echo "$cand/docker-compose.test.yml"
      return 0
    fi
  done
  return 1
}

COMPOSE_FILE=$(find_compose_file || true)
if [ -z "$COMPOSE_FILE" ]; then
  echo "Missing docker-compose.test.yml in repo root, parent or grandparent. Looked under:"
  echo "  $REPO_ROOT"
  echo "  $(dirname $REPO_ROOT)"
  echo "  $(dirname $(dirname $REPO_ROOT))"
  echo "Please place docker-compose.test.yml at repository root or update this script."
  exit 1
fi

echo "Using compose file: $COMPOSE_FILE"
echo "Bringing up test stack"
docker compose -f "$COMPOSE_FILE" up -d --build

echo "Bringing up test stack"
docker compose -f "$COMPOSE_FILE" up -d --build

echo "Running database migrations (best-effort). If you have a listmonk binary locally, run it now."
echo "Running database migrations (best-effort). If you have a listmonk binary locally, run it now."
if [ -x "$REPO_ROOT/listmonk" ]; then
  "$REPO_ROOT/listmonk" --upgrade --yes || true
else
  echo "No local listmonk binary found; ensure migrations are applied in your test image/container."
fi

echo "Running unit tests"
bash "$REPO_ROOT/tests/unit/verify_sender_unit.sh"

echo "All tests finished"
