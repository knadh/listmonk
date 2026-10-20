# listmonk E2E tests

Playwright tests for the server-rendered admin UI. See
[Frontend tests](../docs/docs/content/developer-setup.md#frontend-tests) for the
Docker dev workflow, local prerequisites, and configuration options.

The suite **wipes and reinstalls the configured database schema** and stops
running listmonk processes. Use disposable data; in the Docker dev environment,
this is the `listmonk-dev` database.

From the repository root on the host:

```sh
make build-dev-docker
make run-dev-docker
```

Inside the dev container:

```sh
make build-frontend
make build
cd tests
bun install
bun playwright install --with-deps chromium
bun run test
```

Compose configures the database and existing MailHog service automatically.
Rebuild the backend after Go or SQL changes, and reinstall the browser and its
system dependencies when recreating the dev container.

Outside Docker, install the dependencies described in the developer guide.
From `tests/`, use `bun run test` for headless tests or `bun run test:ui` for the
interactive UI.

The rich-text editor regression tests can also run directly against local
frontend source, without starting listmonk or touching a database. From `tests/`:

```sh
bun playwright test --config playwright.richtext.config.js richtext.spec.js
```
