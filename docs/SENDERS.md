# Sender verification and unverified-sender option

This document describes the verification-by-code feature and the environment option to allow unverified senders.

## New API endpoints

- `POST /api/senders`
  - Purpose: create or request verification for a sender (email + optional name).
  - Request JSON example:

```json
{
  "email": "contact@example.com",
  "name": "Contact Name"
}
```

  - Behavior: creates or upserts a `senders` entry with `verified=false` and generates a `verification_code`. A verification email is sent to the provided address.

- `POST /api/senders/verify`
  - Purpose: confirm/verify a previously created sender using the code sent by email.
  - Request JSON example:

```json
{
  "email": "contact@example.com",
  "code": "ABC123xyz"
}
```

  - Behavior: validates the code and sets `verified=true` for the sender if valid.

## Enforcement in transactional sends

- The transactional send endpoint `POST /api/tx` now enforces that the `from_email` exists in the `senders` table and is `verified=true`.
- If the environment variable `LISTMONK_ALLOW_UNVERIFIED_SENDER=true` is set, this enforcement is skipped and senders that are not verified can still be used. This option is intended for testing and should be used with caution in production.

### Environment variables

- `LISTMONK_ALLOW_UNVERIFIED_SENDER` (default: not set / false)
  - When set to `true`, bypasses the verification enforcement and allows sending from any `From` address.

- `LISTMONK_VERIFIED_RETURN_PATH` (optional)
  - When `LISTMONK_ALLOW_UNVERIFIED_SENDER=true`, you may also set this to a verified return-path address to avoid mailbox clients showing "via" headers (useful for testing).

## Local testing and reproduction

Repository contains test scripts under `/tests`:

- `tests/unit/verify_sender_unit.sh`: unit-style shell test that:
  1. attempts to send with an unverified sender (expect rejection),
  2. creates a sender (`POST /api/senders`),
  3. reads the verification email from MailHog,
  4. verifies the sender (`POST /api/senders/verify`),
  5. attempts the transactional send again (expect success).

- `tests/e2e/verify_sender_e2e.sh`: uses `docker compose -f docker-compose.test.yml up -d --build` to start services (Postgres, MailHog, app) and runs the unit test above.

Run locally (example):

```bash
# from repository root (where docker-compose.test.yml is located)
docker compose -f docker-compose.test.yml up -d --build
./listmonk --upgrade --yes    # apply migrations in the container or host
tests/unit/verify_sender_unit.sh
```

If you prefer the full E2E helper:

```bash
tests/e2e/verify_sender_e2e.sh
```

Notes:
- The test scripts expect MailHog at `http://127.0.0.1:8025` and the app at `http://127.0.0.1:9100` by default; override with `MAILHOG_API` and `LISTMONK_URL` environment variables.
- If your CI runner does not allow running Docker-in-Docker or privileged containers, adapt the workflow to use dedicated test runners or provide external services.
