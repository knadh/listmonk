#!/usr/bin/env bash
set -euo pipefail

# Unit-style integration test (shell) for sender verification logic.
# This script assumes a running local Listmonk instance and MailHog:
# - LISTMONK_URL (default: http://127.0.0.1:9100)
# - MAILHOG_API (default: http://127.0.0.1:8025/api/v2/messages)

LISTMONK_URL=${LISTMONK_URL:-http://127.0.0.1:9100}
MAILHOG_API=${MAILHOG_API:-http://127.0.0.1:8025/api/v2/messages}

SENDER_EMAIL=${SENDER_EMAIL:-test-sender-unverified@local.test}
RECIPIENT_EMAIL=${RECIPIENT_EMAIL:-test-recipient@local.test}

echo "Running unit-style sender verification test against $LISTMONK_URL"

cleanup_mailhog() {
  # remove existing MailHog messages for cleanliness (best-effort)
  curl -s -X DELETE "$MAILHOG_API" || true
}

wait_for() {
  local url=$1
  echo -n "Waiting for $url ... "
  for i in {1..30}; do
    if curl -sSf "$url" >/dev/null 2>&1; then
      echo "ok"
      return 0
    fi
    sleep 1
  done
  echo
  echo "Timed out waiting for $url"
  return 1
}

wait_for "$LISTMONK_URL/health" || true
cleanup_mailhog

echo "1) Attempting to send transactional mail with unverified sender (expect rejection)"
TX_PAYLOAD=$(cat <<JSON
{
  "from_email": "$SENDER_EMAIL",
  "from_name": "Unit Tester",
  "subject": "Unit test - unverified",
  "html": "<p>Test unverified sender</p>",
  "to": [{"email":"$RECIPIENT_EMAIL","name":"Recipient"}]
}
JSON
)

HTTP=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$LISTMONK_URL/api/tx" -H "Content-Type: application/json" -d "$TX_PAYLOAD" || true)
echo "Response code: $HTTP"
if [ "$HTTP" -eq 200 ] || [ "$HTTP" -eq 202 ]; then
  echo "ERROR: transactional send succeeded with unverified sender (unexpected)"
  exit 2
fi

echo "2) Creating sender (POST /api/senders) to trigger verification email"
CREATE_PAYLOAD=$(jq -n --arg e "$SENDER_EMAIL" --arg n "Unit Sender" '{email:$e, name:$n}')
curl -s -X POST "$LISTMONK_URL/api/senders" -H "Content-Type: application/json" -d "$CREATE_PAYLOAD" | jq -r '.' || true

echo "Waiting for verification email in MailHog..."
for i in {1..30}; do
  msg=$(curl -s "$MAILHOG_API" | jq -r ".items[] | select(.Content.Headers.To[] | test(\"$SENDER_EMAIL\")) | .Content.Body" 2>/dev/null || true)
  if [ -n "$msg" ]; then
    echo "Found message body"
    break
  fi
  sleep 1
done
if [ -z "$msg" ]; then
  echo "ERROR: verification email not found in MailHog"
  exit 3
fi

# Try to extract a verification code (a sequence of digits/letters) from the body
CODE=$(echo "$msg" | grep -oE "[0-9A-Za-z]{6,128}" | head -n1 || true)
if [ -z "$CODE" ]; then
  echo "WARNING: could not extract a verification code automatically. Showing message for manual extraction:"
  echo "$msg"
  exit 4
fi
echo "Extracted code: $CODE"

echo "3) Verifying sender with code"
VERIFY_PAYLOAD=$(jq -n --arg e "$SENDER_EMAIL" --arg c "$CODE" '{email:$e, code:$c}')
curl -s -X POST "$LISTMONK_URL/api/senders/verify" -H "Content-Type: application/json" -d "$VERIFY_PAYLOAD" | jq -r '.' || true

echo "4) Attempting transactional send again (expect success)"
HTTP2=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$LISTMONK_URL/api/tx" -H "Content-Type: application/json" -d "$TX_PAYLOAD" || true)
echo "Response code: $HTTP2"
if [ "$HTTP2" -ne 200 ] && [ "$HTTP2" -ne 202 ]; then
  echo "ERROR: transactional send failed after verification"
  exit 5
fi

echo "Unit-style sender verification test: SUCCESS"
