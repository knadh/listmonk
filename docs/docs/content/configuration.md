# Configuration

### TOML Configuration file
One or more TOML files can be read by passing `--config config.toml` multiple times. Apart from a few low level configuration variables and the database configuration, all other settings can be managed from the `Settings` dashboard on the admin UI.

To generate a new sample configuration file, run `--listmonk --new-config`

### Environment variables
Variables in config.toml can also be provided as environment variables prefixed by `LISTMONK_` with periods replaced by `__` (double underscore). Example:

| **Environment variable**       | Example value  |
|--------------------------------|----------------|
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
[Read this](../templating/#system-templates)


### HTTP routes
When configuring auth proxies and web application firewalls, use this table.

#### Private admin endpoints.

| Methods | Route              | Description             |
|---------|--------------------|-------------------------|
| `*`     | `/api/*`           | Admin APIs              |
| `GET`   | `/admin/*`         | Admin UI and HTML pages |
| `POST`  | `/webhooks/bounce` | Admin bounce webhook    |


#### Public endpoints to expose to the internet.

| Methods     | Route                 | Description                                   |
|-------------|-----------------------|-----------------------------------------------|
| `GET, POST` | `/subscription/*`     | HTML subscription pages                       |
| `GET, `     | `/link/*`             | Tracked link redirection                      |
| `GET`       | `/campaign/*`         | Pixel tracking image                          |
| `GET`       | `/public/*`           | Static files for HTML subscription pages      |
| `POST`      | `/webhooks/service/*` | Bounce webhook endpoints for AWS and Sendgrid |


## Media Uploads

### Filesystem

When configuring `docker` volume mounts for using filesystem media uploads, you can follow either of two approaches.

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
    volumes:
      - /data/uploads:/listmonk/uploads
```

The files will be available inside `/data/uploads` directory on the host machine.
