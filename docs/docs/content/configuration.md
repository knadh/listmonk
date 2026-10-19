# Configuration

### TOML Configuration file
One or more TOML files can be read by passing `--config config.toml` multiple times. Apart from a few low level configuration variables and the database configuration, all other settings can be managed from the `Settings` dashboard on the admin UI.

To generate a new sample configuration file, run `listmonk --new-config`

### Environment variables
Variables defined in config.toml can also be provided as environment variables prefixed by `LISTMONK_` with periods replaced by `__` (double underscore). To start listmonk purely with environment variables without a configuration file, set the environment variables and pass the config flag as `--config=""`.

Supported variables:

| **Environment variable**       | Example value  |
| ------------------------------ | -------------- |
| `LISTMONK_app__address`        | "0.0.0.0:9000" |
| `LISTMONK_db__host`            | db             |
| `LISTMONK_db__port`            | 9432           |
| `LISTMONK_db__user`            | listmonk       |
| `LISTMONK_db__password`        | listmonk       |
| `LISTMONK_db__database`        | listmonk       |
| `LISTMONK_db__ssl_mode`        | disable        |


### Security middleware (optional)
listmonk can run the [Guard Core](https://github.com/Guard-Core/echo-guard) security engine in front of all HTTP routes. It is off by default and nothing is registered unless `[security] enabled = true` in config.toml. It screens requests for attack payloads (SQL injection, XSS, path traversal and so on), applies per-route policies, and bans IPs that misbehave. State is in memory, so bans and counters are per instance and reset on restart.

```toml
[security]
enabled = true
trusted_proxies = ["10.0.0.0/8"]
admin_allowlist = ["203.0.113.0/24"]
```

Each key can also be set as an environment variable, for example `LISTMONK_security__enabled=true`. `config.toml.sample` lists every key with its default.

| Policy         | Routes                                                                              | What it does |
| -------------- | ----------------------------------------------------------------------------------- | ------------ |
| admin          | `/admin/*` incl. the login pages, and `/api/*` (not `/api/public/*` or static files) | `admin_allowlist`, `block_cloud_providers`. The `query` param (raw SQL expressions) and request bodies are not scanned. |
| login          | `POST` `/admin/login`, `/admin/login/twofa`, `/admin/forgot`, `/admin/reset`, `/auth/oidc` | `login_rate_limit` per `login_rate_window`, plus the admin policy's allowlist and cloud blocking. |
| subscription   | `POST /subscription/form`, `POST /api/public/subscription`                          | `subscription_rate_limit` per window and `block_cloud_providers`. `POST /subscription/form` also has the honeypot. |
| tracking       | `/link/*`, `/campaign/*/px.png`                                                     | Off unless `tracking_usage_limit` is set. 404s here do not count toward `scanner_404_limit`. |

Everything else gets the global settings only (`allowlist`, `blocklist`, countries, `blocked_user_agents`, `rate_limit`, `emergency_mode`, `scanner_404_limit` and so on).

The honeypot is the hidden `nonce` field that listmonk's subscription forms already have. A bot that fills it is banned on the first hit. Set `honeypot_field = ""` to turn it off.

If listmonk runs behind a reverse proxy, set `trusted_proxies` to the proxy addresses. Without it every request comes from the proxy's IP, and a ban or rate limit would hit all your users at once.

!!! note

    Link and pixel tracking requests often come from a few shared IPs: Gmail fetches pixels through Google's image proxy and Microsoft Safe Links prefetches links. For this reason the tracking policy never blocks cloud providers, and `tracking_usage_action` defaults to `log`. Do not set it to `ban` unless you know your traffic, or every recipient behind those IPs loses opens and clicks.


### Customizing system templates
See [system templates](templating.md#system-templates).


### HTTP routes
When configuring auth proxies and web application firewalls, use this table.

#### Private admin endpoints.

| Methods | Route              | Description             |
| ------- | ------------------ | ----------------------- |
| `*`     | `/api/*`           | Admin APIs              |
| `GET`   | `/admin/*`         | Admin UI and HTML pages |
| `POST`  | `/webhooks/bounce` | Admin bounce webhook    |


#### Public endpoints to expose to the internet.

| Methods     | Route                 | Description                                   |
| ----------- | --------------------- | --------------------------------------------- |
| `GET, POST` | `/subscription/*`     | HTML subscription pages                       |
| `GET, `     | `/link/*`             | Tracked link redirection                      |
| `GET`       | `/campaign/*`         | Pixel tracking image                          |
| `GET`       | `/public/*`           | Static files for HTML subscription pages      |
| `POST`      | `/webhooks/service/*` | Bounce webhook endpoints for SES, Azure ACS, Sendgrid, and other supported providers |
| `GET`       | `/uploads/*`          | The file upload path configured in media settings |


## Media uploads

#### Using filesystem

When configuring `docker` volume mounts for using filesystem media uploads, you can follow either of two approaches. [The second option may be necessary if](https://github.com/knadh/listmonk/issues/1169#issuecomment-1674475945) your setup requires you to use `sudo` for docker commands. 

After making any changes you will need to run `sudo docker compose stop ; sudo docker compose up`. 

And under `https://listmonk.mysite.com/admin/settings` you put `/listmonk/uploads`. 

#### Using volumes

Using `docker volumes`, you can specify the name of volume and destination for the files to be uploaded inside the container.


```yml
app:
    volumes:
      - type: volume
        source: listmonk-uploads
        target: /listmonk/uploads

volumes:
  listmonk-uploads:
```

!!! note

    This volume is managed by `docker` itself, and you can see find the host path with `docker volume inspect listmonk_listmonk-uploads`.

#### Using bind mounts

```yml
  app:
    volumes:
      - ./path/on/your/host/:/path/inside/container
```
Eg:
```yml
  app:
    volumes:
      - ./data/uploads:/listmonk/uploads
```
The files will be available inside `/data/uploads` directory on the host machine.

To use the default `uploads` folder:
```yml
  app:
    volumes:
      - ./uploads:/listmonk/uploads
```

## Logs

### Docker

https://docs.docker.com/engine/reference/commandline/logs/
```
sudo docker logs -f
sudo docker logs listmonk_app -t
sudo docker logs listmonk_db -t
sudo docker logs --help
```
Container info: `sudo docker inspect listmonk_listmonk`

Docker logs to `/dev/stdout` and `/dev/stderr`. The logs are collected by the docker daemon and stored in your node's host path (by default). The same can be configured (/etc/docker/daemon.json) in your docker daemon settings to setup other logging drivers, logrotate policy and more, which you can read about [here](https://docs.docker.com/config/containers/logging/configure/).

### Binary

listmonk logs to `stdout`, which is usually not saved to any file. To save listmonk logs to a file use `./listmonk > listmonk.log`.

Settings -> Logs in admin shows the last 1000 lines of the standard log output but gets erased when listmonk is restarted.

For the [service file](https://github.com/knadh/listmonk/blob/master/listmonk%40.service), you can use `ExecStart=/bin/bash -ce "exec /usr/bin/listmonk --config /etc/listmonk/config.toml --static-dir /etc/listmonk/static >>/etc/listmonk/listmonk.log 2>&1"` to create a log file that persists after restarts. [More info](https://github.com/knadh/listmonk/issues/1462#issuecomment-1868501606).


## Time zone

To change listmonk's time zone (logs, etc.) edit `docker-compose.yml`:
```
environment:
    - TZ=Etc/UTC
```
with any Timezone listed [here](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones). Then run `sudo docker-compose stop ; sudo docker-compose up` after making changes.

## SMTP

### Retries
The `Settings -> SMTP -> Retries` denotes the number of times a message that fails at the moment of sending is retried silently using different connections from the SMTP pool. The messages that fail even after retries are the ones that are logged as errors and ignored.

## SMTP ports
Some server hosts block outgoing SMTP ports (25, 465). You may have to contact your host to unblock them before being able to send e-mails. Eg: [Hetzner](https://docs.hetzner.com/cloud/servers/faq/#why-can-i-not-send-any-mails-from-my-server).

## HELO hostname
By default, listmonk will auto-generate a hostname and advertise that to the SMTP server. Certain hosts such as Google Workspace SMTP and relay require it to be set to an valid hostname (FQDN).


## Performance

### Batch size

The batch size parameter is useful when working with very large lists with millions of subscribers for maximising throughput. It is the number of subscribers that are fetched from the database sequentially in a single cycle (~5 seconds) when a campaign is running. Increasing the batch size uses more memory, but reduces the round trip to the database.
