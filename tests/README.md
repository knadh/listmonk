# listmonk E2E tests

Playwright UI E2E tests. Requires a built `./listmonk` binary in the repo root with a test Postgres instance (use docker-compose.yml) set in config.toml. Every test run wipes and reinstalls the DB schema.

## Setup

```sh
cd tests
bun install
bun playwright install chromium
```

## Run

```sh
bun run test      # runs tests in headless mode
bun run test:ui   # Shows the Playwright UI
```
