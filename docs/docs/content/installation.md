# Installation

listmonk requires Postgres ⩾ 12.

See the "[Tutorials](#tutorials)" section at the bottom for detailed guides. 

## Binary
- Download the [latest release](https://github.com/knadh/listmonk/releases) and extract the listmonk binary.
- `./listmonk --new-config` to generate config.toml. Then, edit the file.
- `./listmonk --install` to install the tables in the Postgres DB.
- Run `./listmonk` and visit `http://localhost:9000`.


## Docker

The latest image is available on DockerHub at `listmonk/listmonk:latest`

!!! note
    Listmonk's docs and scripts use `docker compose`, which is compatible with the latest version of docker. If you installed docker and docker-compose from your Linux distribution, you probably have an older version and will need to use the `docker-compose` command instead, or you'll need to update docker manually. [More info](https://gist.github.com/MaximilianKohler/e5158fcfe6de80a9069926a67afcae11#docker-update).

Use the sample [docker-compose.yml](https://github.com/knadh/listmonk/blob/master/docker-compose.yml) to run listmonk and Postgres DB with `docker compose` as follows:

### Demo

#### Easy Docker install

```bash
mkdir listmonk-demo
sh -c "$(curl -fsSL https://raw.githubusercontent.com/knadh/listmonk/master/install-demo.sh)"
```

#### Manual Docker install

```bash
wget -O docker-compose.yml https://raw.githubusercontent.com/knadh/listmonk/master/docker-compose.yml
docker compose up -d demo-db demo-app
```

!!! warning
    The demo does not persist Postgres after the containers are removed. **DO NOT** use this demo setup in production.

### Production

#### Easy Docker install

This setup is recommended if you want to _quickly_ setup `listmonk` in production.

```bash
mkdir listmonk
sh -c "$(curl -fsSL https://raw.githubusercontent.com/knadh/listmonk/master/install-prod.sh)"
```

The above shell script performs the following actions:

- Downloads `docker-compose.yml` and generates a `config.toml`.
- Runs a Postgres container and installs the database schema.
- Runs the `listmonk` container.

!!! note
    It's recommended to examine the contents of the shell script, before running in your environment.

#### Manual Docker install

The following workflow is recommended to setup `listmonk` manually using `docker compose`. You are encouraged to customise the contents of `docker-compose.yml` to your needs. The overall setup looks like:

- `docker compose up db` to run the Postgres DB.
- `docker compose run --rm app ./listmonk --install` to setup the DB (or `--upgrade` to upgrade an existing DB).
- Copy `config.toml.sample` to your directory and make the following changes:
    - `app.address` => `0.0.0.0:9000` (Port forwarding on Docker will work only if the app is advertising on all interfaces.)
    - `db.host` => `listmonk_db` (Container Name of the DB container)
- Run `docker compose up app` and visit `http://localhost:9000`.

##### Mounting a custom config.toml

To mount a local `config.toml` file, add the following section to `docker-compose.yml`:

```yml
  app:
    <<: *app-defaults
    depends_on:
      - db
    volumes:
    - ./path/on/your/host/config.toml:/listmonk/config.toml
```

!!! note
    Some common changes done inside `config.toml` for Docker based setups:

    - Change `app.address` to `0.0.0.0:9000`.
    - Change `db.host` to `listmonk_db`.

Here's a sample `config.toml` you can use:

```toml
[app]
address = "0.0.0.0:9000"
admin_username = "listmonk"
admin_password = "listmonk"

# Database.
[db]
host = "listmonk_db"
port = 5432
user = "listmonk"
password = "listmonk"
database = "listmonk"
ssl_mode = "disable"
max_open = 25
max_idle = 25
max_lifetime = "300s"
```

Mount the local `config.toml` inside the container at `listmonk/config.toml`.

!!! tip
    - See [configuring with environment variables](../configuration) for variables like `app.admin_password` and `db.password`
    - Ensure that both `app` and `db` containers are in running. If the containers are not running, restart them `docker compose restart app db`.
    - Refer to [this tutorial](https://yasoob.me/posts/setting-up-listmonk-opensource-newsletter-mailing/) for setting up a production instance with Docker + Nginx + LetsEncrypt SSL.

!!! info
    The example `docker-compose.yml` file works with Docker Engine 24.0.5+ and Docker Compose version v2.20.2+.

## Compiling from source

To compile the latest unreleased version (`master` branch):

1. Make sure `go`, `nodejs`, and `yarn` are installed on your system.
2. `git clone git@github.com:knadh/listmonk.git`
3. `cd listmonk && make dist`. This will generate the `listmonk binary`.

## Release candidate (RC)

The `master` branch with bleeding edge changes is periodically built and published as `listmonk/listmonk:rc` on DockerHub. To run the latest pre-release version, replace all instances of `listmonk/listmonk:latest` with `listmonk/listmonk:rc` in the docker-compose.yml file and follow the Docker installation steps above. While it is generally safe to run release candidate versions, they may have issues that only get resolved in a general release.

## 3rd party hosting

<a href="https://railway.app/new/template/listmonk"><img src="https://camo.githubusercontent.com/081df3dd8cff37aab35044727b02b94a8e948052487a8c6253e190f5940d776d/68747470733a2f2f7261696c7761792e6170702f627574746f6e2e737667" alt="One-click deploy on Raleway" style="max-height: 32px;" /></a>
<br />
<a href="https://www.pikapods.com/pods?run=listmonk"><img src="https://www.pikapods.com/static/run-button.svg" alt="Deploy on PikaPod" /></a>


## Tutorials

* [Tutorial for deploying on **Fly.io**](https://github.com/paulrudy/listmonk-on-fly)
* [Complete Listmonk setup guide. Step-by-step tutorial for installation and all basic functions. **Amazon EC2 & SES**](https://gist.github.com/MaximilianKohler/e5158fcfe6de80a9069926a67afcae11)
* [Step-by-step guide on how install and set up Listmonk on a server (**AWS Lightsail**)](https://github.com/knadh/listmonk/issues/1208)
* [Informal step-by-step on how to get started with Listmonk using **Railway**](https://github.com/knadh/listmonk/issues/120#issuecomment-1421838533)
* [**Binary** install on your PC](https://www.youtube.com/watch?v=fAOBqgR9Yfo). Discussions of limitations: [[1](https://github.com/knadh/listmonk/issues/862#issuecomment-1307328228)][[2](https://github.com/knadh/listmonk/issues/248#issuecomment-1320806990)]. 
