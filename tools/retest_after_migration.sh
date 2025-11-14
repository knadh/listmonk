#!/usr/bin/env bash
set -euo pipefail

# retest_after_migration.sh
# Usage: ./tools/retest_after_migration.sh
# This script attempts to restart the test stack, run the DB upgrade inside the app container,
# and execute the local CI helper (`src/scripts/run_local_ci.sh`). Designed for maintainer/CI use.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILES=("${ROOT_DIR}/../docker-compose.test.yml" "${ROOT_DIR}/docker-compose.test.yml" "docker-compose.test.yml")

find_compose_file() {
  for f in "${COMPOSE_FILES[@]}"; do
    if [ -f "$f" ]; then
      echo "$f"
      return 0
    fi
  done
  return 1
}

COMPOSE_FILE=$(find_compose_file) || {
  echo "ERROR: could not find docker-compose.test.yml; please run from repo root or edit the script." >&2
  exit 2
}

echo "Using compose file: $COMPOSE_FILE"

echo "Bringing up test stack (build if necessary)..."
docker compose -f "$COMPOSE_FILE" up -d --build

echo "Attempting to run Listmonk migrations inside the 'app' service..."
if docker compose -f "$COMPOSE_FILE" ps --services | grep -q "app"; then
  # Preferred: exec into running container
  if docker compose -f "$COMPOSE_FILE" exec -T app /listmonk --upgrade --yes; then
    echo "Upgrade command executed with docker compose exec."
  else
    echo "docker compose exec failed; trying a one-off run..."
    docker compose -f "$COMPOSE_FILE" run --rm app /listmonk --upgrade --yes
  fi
else
  echo "No running 'app' service found; running one-off upgrade..."
  docker compose -f "$COMPOSE_FILE" run --rm app /listmonk --upgrade --yes
fi

echo "Upgrade complete (or attempted). Running local CI helper to execute tests..."
if [ -x "$ROOT_DIR/src/scripts/run_local_ci.sh" ]; then
  bash "$ROOT_DIR/src/scripts/run_local_ci.sh"
else
  echo "run_local_ci.sh not found or not executable at: $ROOT_DIR/src/scripts/run_local_ci.sh" >&2
  exit 3
fi

echo "Retest finished. Check the compose logs and test outputs for details."

exit 0
