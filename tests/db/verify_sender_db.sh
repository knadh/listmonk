#!/usr/bin/env bash
set -euo pipefail

# DB-level verification test for senders table.
# Connects to the postgres container and performs simple checks:
# - inserts an unverified sender
# - confirms verified=false
# - sets verified=true
# - confirms verified=true

DB_CONTAINER=${DB_CONTAINER:-listmonk_test_db}
DB_USER=${DB_USER:-listmonk}
DB_NAME=${DB_NAME:-listmonk}

SENDER_EMAIL=${SENDER_EMAIL:-db-test-sender@local.test}

echo "Running DB-level sender verification test against container $DB_CONTAINER"

run_sql() {
  local sql="$1"
  docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -t -A -c "$sql"
}

echo "1) Ensure senders table exists"
run_sql "CREATE TABLE IF NOT EXISTS senders (id SERIAL PRIMARY KEY, email VARCHAR(255) NOT NULL, name VARCHAR(255), verified BOOLEAN DEFAULT false, verification_code VARCHAR(128), created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now()); CREATE UNIQUE INDEX IF NOT EXISTS idx_senders_email_lower ON senders (LOWER(email));"

echo "2) Insert or upsert unverified sender"
run_sql "INSERT INTO senders (email, name, verified) VALUES ('${SENDER_EMAIL}', 'DB Test Sender', false) ON CONFLICT (LOWER(email)) DO UPDATE SET name=EXCLUDED.name, verified=EXCLUDED.verified RETURNING id;"

echo "3) Confirm verified=false"
VER=$(run_sql "SELECT verified FROM senders WHERE LOWER(email)=LOWER('${SENDER_EMAIL}') LIMIT 1;")
echo "verified value: '$VER'"
if [ "$VER" != "f" ] && [ "$VER" != "false" ]; then
  echo "ERROR: expected verified=false, got '$VER'"
  exit 1
fi

echo "4) Set verified=true"
run_sql "UPDATE senders SET verified=true WHERE LOWER(email)=LOWER('${SENDER_EMAIL}');"

echo "5) Confirm verified=true"
VER2=$(run_sql "SELECT verified FROM senders WHERE LOWER(email)=LOWER('${SENDER_EMAIL}') LIMIT 1;")
echo "verified value after update: '$VER2'"
if [ "$VER2" != "t" ] && [ "$VER2" != "true" ]; then
  echo "ERROR: expected verified=true, got '$VER2'"
  exit 1
fi

echo "DB-level sender verification test: SUCCESS"
