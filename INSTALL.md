# Install and run

- Run `./listmonk --new-config` to generate a sample `config.toml` and add your configuration (SMTP and Postgres DB credentials primarily).
- `./listmonk --install` to setup the DB.
- Run `./listmonk` and visit `http://localhost:9000`.

## Running on Docker

You can checkout the [docker-compose.yml](docker-compose.yml) to get an idea of how to run `listmonk` with `PostgreSQL` together using Docker.

- **Run the services**: `docker-compose up -d app db` to run all the services together. If this is a first time setup, you will see some errors related to DB which occur because migrations haven't been applied yet. Don't worry, follow the next step.
- **Apply DB migrations**: `docker-compose run --rm app ./listmonk --install`.
-  Ensure that both the containers are in running state before proceeding. If the app container is not `up`, you might need to restart the app container once: `docker-compose restart app`. 
- Visit `http://localhost:9000`.

### Mounting a custom config file

You are expected to tweak [config.toml.sample](config.toml.sample) for actual use with your custom settings. To mount the `config.toml` file,
you can add the following section to `docker-compose.yml`:

```
  app:
    <<: *app-defaults
    depends_on:
      - db
    volume:
    - ./path/on/host/config.toml/:/listmonk/config.toml
```

This will `mount` your local `config.toml` inside the container at `listmonk/config.toml`.

_NOTE_: This `docker-compose` file works with Docker Engine 18.06.0+ and `docker-compose` which supports file format 3.7.

### Demo Setup

`docker-compose.yml` includes a demo setup to quickly try out `listmonk`. It spins up PostgreSQL and listmonk app containers without any persistent data.

- Run `docker-compose up -d demo-db demo-app`.
- Visit `http://localhost:9000`.

_NOTE_: This setup will delete the data once you kill and remove the containers. This setup is NOT intended for production use.
