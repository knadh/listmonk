#!/usr/bin/env bash
set -euo pipefail

# End-to-end test that drives the full verification flow.
# Assumptions:
# - The Listmonk app is reachable at $LISTMONK_URL (default: http://127.0.0.1:9100)
# - MailHog is reachable at $MAILHOG_API (default: http://127.0.0.1:8025/api/v2/messages)
# - Docker Compose file `docker-compose.test.yml` exists at the repository root and can bring up services.

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
LISTMONK_URL=${LISTMONK_URL:-http://127.0.0.1:9100}
MAILHOG_API=${MAILHOG_API:-http://127.0.0.1:8025/api/v2/messages}

echo "Running full E2E verification flow"

echo "Starting docker-compose test stack (will build images if needed)"
docker compose -f "$REPO_ROOT/docker-compose.test.yml" up -d --build

echo "Waiting for Listmonk to be ready..."
for i in {1..60}; do
  if curl -sSf "$LISTMONK_URL/health" >/dev/null 2>&1; then
    echo "Listmonk ready"
    break
  fi
  sleep 2
done

echo "Running unit-style test as part of E2E"
bash "$REPO_ROOT/tests/unit/verify_sender_unit.sh"

echo "E2E verification flow completed." 
