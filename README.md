<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" /></a>

[![listmonk-logo](https://user-images.githubusercontent.com/547147/231084896-835dba66-2dfe-497c-ba0f-787564c0819e.png)](https://listmonk.app)

listmonk is a standalone, self-hosted, newsletter and mailing list manager. It is fast, feature-rich, and packed into a single binary. It uses a PostgreSQL (⩾ 12) database as its data store.

[![listmonk-dashboard](https://user-images.githubusercontent.com/547147/134939475-e0391111-f762-44cb-b056-6cb0857755e3.png)](https://listmonk.app)

Visit [listmonk.app](https://listmonk.app) for more info. Check out the [**live demo**](https://demo.listmonk.app).

## Installation

### Docker

The latest image is available on DockerHub at [`listmonk/listmonk:latest`](https://hub.docker.com/r/listmonk/listmonk/tags?page=1&ordering=last_updated&name=latest). Use the sample [docker-compose.yml](https://github.com/knadh/listmonk/blob/master/docker-compose.yml) to run manually or use the helper script. 

#### Demo

```bash
mkdir listmonk-demo && cd listmonk-demo
bash -c "$(curl -fsSL https://raw.githubusercontent.com/knadh/listmonk/master/install-demo.sh)"
```

DO NOT use this demo setup in production.

#### Production

```bash
mkdir listmonk && cd listmonk
bash -c "$(curl -fsSL https://raw.githubusercontent.com/knadh/listmonk/master/install-prod.sh)"
```
Visit `http://localhost:9000`.

**NOTE**: Always examine the contents of shell scripts before executing them.

See [installation docs](https://listmonk.app/docs/installation).

__________________

### Binary
- Download the [latest release](https://github.com/knadh/listmonk/releases) and extract the listmonk binary.
- `./listmonk --new-config` to generate config.toml. Then, edit the file.
- `./listmonk --install` to setup the Postgres DB (or `--upgrade` to upgrade an existing DB. Upgrades are idempotent and running them multiple times have no side effects).
- Run `./listmonk` and visit `http://localhost:9000`.

See [installation docs](https://listmonk.app/docs/installation).
__________________


## Developers
listmonk is a free and open source software licensed under AGPLv3. If you are interested in contributing, refer to the [developer setup](https://listmonk.app/docs/developer-setup). The backend is written in Go and the frontend is Vue with Buefy for UI. 


## License
listmonk is licensed under the AGPL v3 license.
