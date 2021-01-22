<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" /></a>

![listmonk](https://user-images.githubusercontent.com/547147/89733021-43fbf700-da70-11ea-82e4-e98cb5010257.png)

listmonk is a standalone, self-hosted, newsletter and mailing list manager. It is fast, feature-rich, and packed into a single binary. It uses a PostgreSQL database as its data store.

[![listmonk-dashboard](https://user-images.githubusercontent.com/547147/89733057-87566580-da70-11ea-8160-855f6f046a55.png)](https://listmonk.app)
Visit [listmonk.app](https://listmonk.app)

## Installation

### Docker

The latest image is available on DockerHub at `listmonk/listmonk:latest`. Use the sample [docker-compose.yml](https://github.com/knadh/listmonk/blob/master/docker-compose.yml) to run listmonk and Postgres DB with docker-compose as follows:

#### Demo

```bash
mkdir listmonk-demo
sh -c "$(curl -sSL https://raw.githubusercontent.com/knadh/listmonk/master/install-demo.sh)"
```

The demo does not persist Postgres after the containers are removed. DO NOT use this demo setup in production.

#### Production
- `docker-compose up db` to run the Postgres DB.
- `docker-compose run --rm app ./listmonk --install` to setup the DB (or `--upgrade` to upgrade an existing DB)
- Run `docker-compose up app` and visit `http://localhost:9000`.

More information on [docs](https://listmonk.app/docs).

__________________

### Binary
- Download the [latest release](https://github.com/knadh/listmonk/releases) and extract the listmonk binary.
- `./listmonk --new-config` to generate config.toml. Then, edit the file.
- `./listmonk --install` to setup the Postgres DB (or `--upgrade` to upgrade an existing DB. Upgrades are idempotent and running them multiple times have no side effects).
- Run `./listmonk` and visit `http://localhost:9000`.

__________________

### Heroku 

Using the [Nginx buildpack](https://github.com/heroku/heroku-buildpack-nginx) can be used to deploy listmonk on Heroku and use Nginx as a proxy to setup basicauth. 
This one-click [Heroku deploy button](https://github.com/bumi/listmonk-heroku) provides an automated default deployment.

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/bumi/listmonk-heroku)

Please note that [configuration options](https://listmonk.app/docs/configuration) must be set using [environment configuration variables](https://devcenter.heroku.com/articles/config-vars).



## Developers
listmonk is a free and open source software licensed under AGPLv3. If you are interested in contributing, refer to the [developer setup](https://listmonk.app/docs/developer-setup). The backend is written in Go and the frontend is Vue with Buefy for UI. 


## License
listmonk is licensed under the AGPL v3 license.
