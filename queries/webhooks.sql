-- webhooks

-- name: create-webhook-log
-- Creates a new webhook log entry with triggered status.
INSERT INTO webhook_logs (webhook_id, event, payload, status, created_at, updated_at)
VALUES ($1, $2, $3, 'triggered', NOW(), NOW())
RETURNING id;

-- name: get-pending-webhook-logs
-- Fetches a batch of triggered webhook logs and locks them for processing.
-- Uses SKIP LOCKED to allow concurrent workers to process different batches.
UPDATE webhook_logs
SET status = 'processing', updated_at = NOW()
WHERE id IN (
    SELECT id FROM webhook_logs
    WHERE status = 'triggered'
    ORDER BY created_at ASC
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
RETURNING id, webhook_id, event, payload, retries, created_at, updated_at;

-- name: update-webhook-log-success
-- Marks a webhook log as completed with response data.
UPDATE webhook_logs
SET status = 'completed',
    response = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: update-webhook-log-retry
-- Updates a webhook log after a failed attempt, incrementing retry count.
UPDATE webhook_logs
SET retries = retries + 1,
    last_retried_at = NOW(),
    response = $2,
    note = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: update-webhook-log-failed
-- Marks a webhook log as failed after all retries exhausted.
UPDATE webhook_logs
SET status = 'failed',
    response = $2,
    note = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: mark-webhook-log-triggered
-- Resets a processing webhook log back to triggered status (for recovery after crash).
UPDATE webhook_logs
SET status = 'triggered',
    updated_at = NOW()
WHERE id = $1;

-- name: reset-stale-processing-logs
-- Resets webhook logs that have been stuck in processing status for too long (recovery after crash).
-- This should be called on app startup.
UPDATE webhook_logs
SET status = 'triggered',
    updated_at = NOW()
WHERE status = 'processing'
  AND updated_at < NOW() - INTERVAL '5 minutes';

-- name: delete-old-webhook-logs
-- Deletes old completed and failed webhook logs older than specified days.
DELETE FROM webhook_logs
WHERE status IN ('completed', 'failed')
  AND created_at < NOW() - ($1 || ' days')::INTERVAL;
