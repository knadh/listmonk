# Installation

listmonk is a simple binary application that requires a Postgres database instance to run. The binary can be downloaded and run manually, or it can be run as a container with Docker compose.

## Binary
1. Download the [latest release](https://github.com/knadh/listmonk/releases) and extract the listmonk binary. `amd64` is the main one. It works for Intel and x86 CPUs.
1. `./listmonk --new-config` to generate config.toml. Edit the file.
1. `./listmonk --install` to install the tables in the Postgres DB (⩾ 12).
1. Run `./listmonk` and visit `http://localhost:9000` to create the Super Admin user and login.

!!! Tip
    To set the Super Admin username and password during installation, set the environment variables:
    `LISTMONK_ADMIN_USER=myuser LISTMONK_ADMIN_PASSWORD=xxxxx ./listmonk --install`


## Docker

The latest image is available on DockerHub at `listmonk/listmonk:latest`

The recommended method is to download the [docker-compose.yml](https://github.com/knadh/listmonk/blob/master/docker-compose.yml) file, customize it for your environment and then to simply run `docker compose up -d`.

```shell
# Download the compose file to the current directory.
curl -LO https://github.com/knadh/listmonk/raw/master/docker-compose.yml

# Run the services in the background.
docker compose up -d
```

Then, visit `http://localhost:9000` to create the Super Admin user and login.

!!! Tip
    To set the Super Admin username and password during setup, set the environment variables (only the first time):
    `LISTMONK_ADMIN_USER=myuser LISTMONK_ADMIN_PASSWORD=xxxxx docker compose up -d`


------------

### Mounting a custom config.toml
The docker-compose file includes all necessary listmonk configuration as environment variables, `LISTMONK_*`.
If you would like to remove those and mount a config.toml instead:

#### 1. Save the config.toml file on the host

```toml
[app]
address = "0.0.0.0:9000"

# Database.
[db]
host = "listmonk_db" # Postgres container name in the compose file.
port = 5432
user = "listmonk"
password = "listmonk"
database = "listmonk"
ssl_mode = "disable"
max_open = 25
max_idle = 25
max_lifetime = "300s"
```

#### 2. Mount the config file in docker-compose.yml

```yaml
  app:
    ...
    volumes:
    - /path/on/your/host/config.toml:/listmonk/config.toml
```

#### 3. Change the `--config ''` flags in the `command:` section to point to the path

```yaml
command: [sh, -c, "./listmonk --install --idempotent --yes --config /listmonk/config.toml && ./listmonk --upgrade --yes --config /listmonk/config.toml && ./listmonk --config /listmonk/config.toml"]
```


## Compiling from source

To compile the latest unreleased version (`master` branch):

1. Make sure `go`, `nodejs`, and `yarn` are installed on your system.
2. `git clone git@github.com:knadh/listmonk.git`
3. `cd listmonk && make dist`. This will generate the `listmonk` binary.

## Release candidate (RC)

The `master` branch with bleeding edge changes is periodically built and published as `listmonk/listmonk:rc` on DockerHub. To run the latest pre-release version, replace all instances of `listmonk/listmonk:latest` with `listmonk/listmonk:rc` in the docker-compose.yml file and follow the Docker installation steps above. While it is generally safe to run release candidate versions, they may have issues that only get resolved in a general release.

## Helm chart for Kubernetes

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 3.0.0](https://img.shields.io/badge/AppVersion-3.0.0-informational?style=flat-square)

A helm chart for easily installing listmonk on a kubernetes cluster is made available by community [here](https://github.com/th0th/helm-charts/tree/main/charts/listmonk).

In order to use the helm chart, you can configure `values.yaml` according to your needs, and then run the following command:

```shell
$ helm upgrade \
    --create-namespace \
    --install listmonk listmonk \
    --namespace listmonk \
    --repo https://th0th.github.io/helm-charts \
    --values values.yaml \
    --version 0.1.0
```

## 3rd party hosting

<a href="https://dash.elest.io/deploy?soft=Listmonk&id=237"><img src="https://raw.githubusercontent.com/elestio-examples/reactjs/refs/heads/master/src/deploy-on-elestio.png" alt="Deploy to Elestio" height="35" style="max-width: 150px;" /></a>
<br />
<a href="https://www.pikapods.com/pods?run=listmonk"><img src="https://www.pikapods.com/static/run-button.svg" alt="Deploy on PikaPod" style="max-width: 150px;" /></a>
<br />
<a href="https://railway.app/new/template/listmonk"><img src="https://railway.app/button.svg" alt="One-click deploy on Railway" style="max-width: 150px;" /></a>
<br />
<a href="https://repocloud.io/details/?app_id=217"><img src="https://d16t0pc4846x52.cloudfront.net/deploy.png" alt="Deploy at RepoCloud" style="max-width: 150px;"/></a>
<br />
<a href="https://template.sealos.io/deploy?templateName=listmonk"><img src="https://sealos.io/Deploy-on-Sealos.svg" alt="Deploy on Sealos" style="max-width: 150px;"/></a>
<br />
<a href="https://zeabur.com/templates/5EDMN6"><img src="https://zeabur.com/button.svg" alt="Deploy on Zeabur" style="max-width: 150px;"/></a>

## Tutorials

* [Informal step-by-step on how to get started with listmonk using *Railway*](https://github.com/knadh/listmonk/issues/120#issuecomment-1421838533)
* [Step-by-step tutorial for installation and all basic functions. *Amazon EC2, SES, docker & binary*](https://gist.github.com/MaximilianKohler/e5158fcfe6de80a9069926a67afcae11)
* [Step-by-step guide on how to install and set up listmonk on *AWS Lightsail with docker* (rameerez)](https://github.com/knadh/listmonk/issues/1208)
* [Quick setup on any cloud server using *docker and caddy*](https://github.com/samyogdhital/listmonk-caddy-reverse-proxy)
* [*Binary* install on Ubuntu 22.04 as a service](https://mumaritc.hashnode.dev/how-to-install-listmonk-using-binary-on-ubuntu-2204)
* [*Binary* install on Ubuntu 18.04 as a service (Apache & Plesk)](https://devgypsy.com/post/2020-08-18-installing-listmonk-newsletter-manager/)
* [*Binary and docker* on linux (techviewleo)](https://techviewleo.com/manage-mailing-list-and-newsletter-using-listmonk/)
* [*Binary* install on your PC](https://www.youtube.com/watch?v=fAOBqgR9Yfo). Discussions of limitations: [[1](https://github.com/knadh/listmonk/issues/862#issuecomment-1307328228)][[2](https://github.com/knadh/listmonk/issues/248#issuecomment-1320806990)].
* [*Docker on Rocky Linux 8* (nginx, Let's Encrypt SSL)](https://wiki.crowncloud.net/?How_to_Install_Listmonk_with_Docker_on_Rocky_Linux_8)
* [*Docker* with nginx reverse proxy, certbot SSL, and Gmail SMTP](https://www.maketecheasier.com/create-own-newsletter-with-listmonk/)
* [Install Listmonk on Self-hosting with *Pre-Configured AMI Package at AWS* by Single Click](https://meetrix.io/articles/how-to-install-llama-2-on-aws-with-pre-configured-ami-package/)
* [Tutorial for deploying on *Fly.io*](https://github.com/paulrudy/listmonk-on-fly) -- Currently [not working](https://github.com/knadh/listmonk/issues/984#issuecomment-1694545255)
