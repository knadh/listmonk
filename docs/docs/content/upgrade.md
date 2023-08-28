# Upgrade

Some versions may require changes to the database. These changes or database "migrations" are applied automatically and safely, but, it is recommended to take a backup of the Postgres database before running the `--upgrade` option, especially if you have made customizations to the database tables.

## Binary
- Download the [latest release](https://github.com/knadh/listmonk/releases) and extract the listmonk binary.
- `./listmonk --upgrade` to upgrade an existing DB. Upgrades are idempotent and running them multiple times have no side effects.
- Run `./listmonk` and visit `http://localhost:9000`.

## Docker

- `docker compose pull` to pull the latest version from DockerHub.
- `docker compose run --rm app ./listmonk --upgrade` to upgrade an existing DB.
- Run `docker compose up app db` and visit `http://localhost:9000`.

## Railway
- Head to your dashboard, and select your Listmonk project.
- Select the GitHub deployment service.
- In the Deployment tab, head to the latest deployment, click on the three vertical dots to the right, and select "Redeploy".

![Railway Redeploy option](https://user-images.githubusercontent.com/55474996/226517149-6dc512d5-f862-46f7-a57d-5e55b781ff53.png)

## Downgrade

To restore a previous version, you have to restore the DB for that particular version. DBs that have been upgraded with a particular version shouldn't be used with older versions. There may be DB changes that a new version brings that are incompatible with previous versions.

**General steps:**

1. Stop listmonk.
2. Restore your pre-upgrade database.
3. If you're using `docker compose`, edit `docker-compose.yml` and change `listmonk:latest` to `listmonk:v2.4.0` _(for example)_.
4. Restart.

**Example with docker:**

1. Stop listmonk (app):
```
sudo docker stop listmonk_app
```
2. Restore your pre-upgrade db (required) _(be careful, this will wipe your existing DB)_:
```
psql -h 127.0.0.1 -p 9432 -U listmonk
drop schema public cascade;
create schema public;
\q
psql -h 127.0.0.1 -p 9432 -U listmonk -W listmonk < listmonk-preupgrade-db.sql
```
3. Edit the `docker-compose.yml`:
```
x-app-defaults: &app-defaults
  restart: unless-stopped
  image: listmonk/listmonk:v2.4.0
```
4. Restart:
`sudo docker compose up -d app db nginx certbot`

