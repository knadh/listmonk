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


### Running the dev environment
You can run your dev environment locally or inside containers.

After setting up the dev environment, you can visit `http://localhost:9000`.


1. Locally

    - Run `make run` to start the listmonk dev server on `:9000`. It builds the SSR admin assets (from `static/admin/`) and serves the admin at `/admin`. To rebuild admin assets on change while developing, run `cd static/admin && bun run watch` in a separate terminal.

2. Inside containers (Using Makefile)

    - Run `make init-dev-docker` to setup container for db.
    - Run `make dev-docker` to setup docker container suite.
    - Run `make rm-dev-docker` to clean up docker container suite.

3. Inside containers (Using devcontainer)

    - Open repo in vscode, open command palette, and select "Dev Containers: Rebuild and Reopen in Container".

It will set up db, and start frontend/backend for you.


# Production build
Run `make dist` to build the SSR admin frontend and the Go binary, embedding the static assets into a single self-contained binary, `listmonk`.
