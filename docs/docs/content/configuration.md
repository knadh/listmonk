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

| Methods     | Route                        | Description                                   |
| ----------- | ---------------------------- | --------------------------------------------- |
| `GET, POST` | `/subscription/*`            | HTML subscription pages                       |
| `GET, `     | `/link/*`                    | Tracked link redirection                      |
| `GET`       | `/campaign/*`                | Pixel tracking image                          |
| `GET`       | `/public/*`                  | Static files for HTML subscription pages      |
| `POST`      | `/webhooks/service/*`        | Bounce webhook endpoints for SES, Azure ACS, Sendgrid, and other supported providers |
| `GET`       | `/uploads/*`                 | The file upload path configured in media settings |
| `GET`       | `/api/public/captcha/altcha` | (if using altcha captcha) The api endpoint for altcha |


### Signed subscriber links

With `security.signed_links.enabled` enabled, unsubscribe, preference, opt-in,
privacy export/deletion, and hosted message URLs require a signature. listmonk generates these links automatically. The signature
covers the URL path and the `exp`, `l` (opt-in list IDs), `manage`, and `auth` query parameters.
Unrelated query parameters, such as UTM tracking tags, can be added, changed, or removed.
Public signup forms and
tracking links/pixels do not require signatures.

**Defaults:** enabled on fresh installs, disabled when upgrading to v7 so existing
links continue to work. Change this in Settings → Security. With signing disabled,
links are generated and accepted without signatures. Enabling it rejects unsigned
links in previously delivered messages; send fresh messages or opt-in confirmations
to provide signed links. Upgrades preserve an existing choice.
Custom public templates must preserve signed URLs: use `.Data.ManageURL`,
`.Data.ExportURL`, and `.Data.WipeURL` in the subscription template, and retain the
current URL (including its query) when submitting subscription and opt-in forms.
Custom opt-in notification templates should use `.ManageURL` instead of appending
`?manage=true` to `.UnsubURL`. Management-link request forms must POST
`action=request-manage` to `.Data.ManageURL`.

A random secret key is created during installation or upgrade in the database setting
`internal.secret_key`. Preserve this setting in backups and across instances;
changing or deleting it invalidates all previously signed links. It is not exposed
through the settings API. A signed subscription link permits campaign unsubscribe
and requesting a management link by email. Preference changes, global blocklisting,
export, and deletion require the separately emailed management link, which expires in an hour.

The `security.signed_links` setting defaults to:
```json
{"enabled": true, "ttl": "120d"}
```
The TTL supports Go duration units and `d` for 24-hour days: `120d` is 120 days; use `0` for no expiry.
Positive durations must be at least one second. Expiry applies to unsubscribe
links too. Each email's links share a signed expiry time, which is preserved when
opening preference pages or hosted messages. Changing the TTL affects new emails
only. With signing enabled, links without a signed expiry are rejected.

#### Secret key
listmonk generates and stores a secret key in the settings table (`internal.secret_key`) when it's first setup. This secret is used to sign links.

To rotate the key, run the following SQL and restart listmonk. This invalidates all previously sent signed links.
```sql
UPDATE settings SET value = TO_JSONB(ENCODE(GEN_RANDOM_BYTES(32), 'hex')) WHERE key = 'internal.secret_key';
```



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
