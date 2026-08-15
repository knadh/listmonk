# listmonk E2E tests

Playwright UI E2E tests. Requires a built `./listmonk` binary in the repo root with a test Postgres instance (use docker-compose.yml) set in config.toml. Every spec wipes and reinstalls the DB schema (see `resetDB` in `helpers.js`).

`resetDB` also points listmonk's SMTP config at [MailHog](https://github.com/mailhog/MailHog) so mail is captured and testable via its API (`getMail`/`clearMail`). It needs `mailhog` and `psql` on `PATH`; DB credentials are read from the `LISTMONK_db__*` env (falling back to the config.toml defaults).

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
