# Developer setup
The app is a Go backend that server-renders the admin UI (HTML templates with JS, built with [bun](https://bun.sh)). 


### Pre-requisites
- `go`
- `bun` (if you are working on the admin frontend)
- Postgres database. If there is no local installation, the demo docker DB can be used for development (`docker compose up demo-db`)


### First time setup
`git clone https://github.com/knadh/listmonk.git`. The project uses go.mod, so it's best to clone it outside the Go src path.

1. Copy `config.toml.sample` as `config.toml` and add your config.
2. Build the frontend and the binary: `make dist` (builds the SSR admin and the Go binary, embedding the assets). Once the binary is built, run `./listmonk --install` to run the DB setup. For subsequent dev runs, use `make run`.

> [mailhog](https://github.com/mailhog/MailHog) is an excellent standalone mock SMTP server (with a UI) for testing and dev.


### Running the Docker dev environment

- Run `make build-dev-docker` once to build the dev container which bundles Go, Postgres, Bun etc.
- Run `make run-dev-docker` to start it and enter its shell.
- Inside the shell, run `make run` which builds static assets, initializes a fresh DB, and starts listmonk on `:9000`.
- For subsequent changes, simply Ctrl-C and re-run `make run` for changes to Go and static files to be picked up.


Run `make stop-dev-docker` to stop the container and `make rm-dev-docker` to completely remove it.

# Production build
Run `make dist` to build the SSR admin frontend and the Go binary, embedding the static assets into a single self-contained binary, `listmonk`.
