# Configuration

### TOML Configuration file
One or more TOML files can be read by passing `--config config.toml` multiple times. Apart from a few low level configuration variables and the database configuration, all other settings can be managed from the `Settings` dashboard on the admin UI.

To generate a new sample configuration file, run `--listmonk --new-config`

### Environment variables
Variables in config.toml can also be provided as environment variables prefixed by `LISTMONK_` with periods replaced by `__` (double underscore). Example:

| **Environment variable**       | Example value  |
| ------------------------------ | -------------- |
| `LISTMONK_app__address`        | "0.0.0.0:9000" |
| `LISTMONK_app__admin_username` | listmonk       |
| `LISTMONK_app__admin_password` | listmonk       |
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

| Methods     | Route                 | Description                                   |
| ----------- | --------------------- | --------------------------------------------- |
| `GET, POST` | `/subscription/*`     | HTML subscription pages                       |
| `GET, `     | `/link/*`             | Tracked link redirection                      |
| `GET`       | `/campaign/*`         | Pixel tracking image                          |
| `GET`       | `/public/*`           | Static files for HTML subscription pages      |
| `POST`      | `/webhooks/service/*` | Bounce webhook endpoints for AWS and Sendgrid |


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

