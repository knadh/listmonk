# Install and run

- Run `./listmonk --new-config` to generate a sample `config.toml` and add your configuration (SMTP and Postgres DB credentials primarily).
- `./listmonk --install` to setup the DB.
- Run `./listmonk` and visit `http://localhost:9000`.

## Running on Docker

You can checkout the [docker-compose.yml](docker-compose.yml) to get an idea of how to run `listmonk` with `PostgreSQL` together using Docker.

- `docker-compose up -d` to run all the services together.
- `docker-compose run --rm app ./listmonk --install` to setup the DB.
- Visit `http://localhost:9000`.

### Demo Setup

`docker-compose.yml` includes a demo setup to quickly try out `listmonk`. It spins up PostgreSQL and listmonk app containers without any persistent data.

- Run `docker-compose up -d demo-db demo-app`.
- Visit `http://localhost:9000`.

_NOTE_: This setup will delete the data once you kill and remove the containers. This setup is NOT intended for production use.
