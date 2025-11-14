# PR Comment Template — ready to post

Hello @maintainers,

I've prepared tests and an evidence pack for PR #2755 which implements sender verification-by-code and an opt-in envelope rewrite to allow sending with unverified senders for testing.

Summary of what I need from you to complete verification:

1. Please run the database migrations in CI or on a runner with adequate permissions:

```bash
docker compose -f docker-compose.test.yml up -d --build
docker compose -f docker-compose.test.yml exec app /listmonk --upgrade --yes
```

2. After migrations complete, re-run the test helper locally (on the same runner) or allow CI to run it. I included a one-click helper:

```bash
bash src/tools/retest_after_migration.sh
```

What I verified locally:
- DB-level tests for `senders` table passed (insertion and toggling of `verified` flag).
- End-to-end HTTP tests are blocked until the migrations are applied (`listmonk --upgrade --yes`).

Logs and artifacts are available in the branch under `src/docs/pr-2755-test-proof.md`. Local logs produced on my test VM are in `/tmp/ci_run.log` and `/tmp/db_verify_run.log` (attached to my review comment previously).

If you'd like, I can re-run the full tests and post fresh logs immediately after you confirm the migrations have been applied on CI/runner. Thanks!

-- Copilot
