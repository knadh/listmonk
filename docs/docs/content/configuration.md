# Configuration

### TOML Configuration file
One or more TOML files can be read by passing `--config config.toml` multiple times. Apart from a few low level configuration variables and the database configuration, all other settings can be managed from the `Settings` dashboard on the admin UI.

To generate a new sample configuration file, run `--listmonk --new-config`

### Environment variables
Variables in config.toml can also be provided as environment variables prefixed by `LISTMONK_` with periods replaced by `__` (double underscore). To start listmonk purely with environment variables without a configuration file, set the environment variables and pass the config flag as `--config=""`.

Example:

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

| Methods     | Route                 | Description                                   |
| ----------- | --------------------- | --------------------------------------------- |
| `GET, POST` | `/subscription/*`     | HTML subscription pages                       |
| `GET, `     | `/link/*`             | Tracked link redirection                      |
| `GET`       | `/campaign/*`         | Pixel tracking image                          |
| `GET`       | `/public/*`           | Static files for HTML subscription pages      |
| `POST`      | `/webhooks/service/*` | Bounce webhook endpoints for AWS and Sendgrid |
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


## Performance

### Batch size

The batch size parameter is useful when working with very large lists with millions of subscribers for maximising throughput. It is the number of subscribers that are fetched from the database sequentially in a single cycle (~5 seconds) when a campaign is running. Increasing the batch size uses more memory, but reduces the round trip to the database.

## Single Sign On

Listmonk has OIDC support to use a Single Sign On system to connect to it.

!!! note "Automatic user creation"

    There is no automatic user creation on first connexion, but this might get changed in the futur. As of May 2025, you need to create a user with the same email address as the SSO user to enable connexion of said user. Please see [this issue for reference.](https://github.com/knadh/listmonk/issues/2209)


### Keycloak

To configure the Keycloak SSO :

1. In Listmonk, under Settings > Settings > Security, switch *Enable OIDC SSO*
    - In **Provider URL**, set `https://<keycloak.example.ch>/auth/realms/<realm name>`
    - In **Provider Name**, set the name of the SSO, this will be shown to users on the connexion button.
    - In **Client ID**, set the name of the client you will use in Keycloak.
    - In **Client Secret**, set the Secret you will get in the next step from Keycloak.
    - You should see the **Redirect URL for oAuth provider** generated below, that you will use in Keycloak.

2. In KeyCloak, create a new Client with the following options
    - **General settings**
        - **Client ID** : Set client name you set in the step before.
    - **Capability config**
          - Client authentication: On
          - Authorization: On
          - Authentication Flow:
              - Standard Flow: On
              - Direct Access grants: On
    - **Login settings** :
        - Root URL: Auto generated from listmonk, should be: `https://<listmonk.example.com>/auth/oidc`
        - Valid redirect URIs: Same as Root URL
        - Valid post logout redirect URIs: `*`

3. Once the client is created, go to the client's settings in Keycloak, and navigate to **Credentials**, copy Client Secret.
4. Use the Client Secret and Client ID to populate Listmonk settings.
5. Hit save
6. Create a user in listmonk with the same email as the one in your SSO
7. Try to login as this user

!!! info "Tracking issue"
    In case you need help with setting up Keycloak, please [see this issue](https://github.com/knadh/listmonk/issues/2209)

### Authentik

1. In Listmonk, under Settings > Settings > Security, switch *Enable OIDC SSO*
    - In **Provider URL**, set `https://<authentik.example.fr>/application/o/<listmonk>/`
    - In **Provider Name**, set the name of the SSO, this will be shown to users on the connexion button.
    - In **Client ID**, set the name of the client you will use in Keycloak.
    - In **Client Secret**, set the Secret you will get in the next step from Keycloak.
    - You should see the **Redirect URL for oAuth provider** generated below, that you will use in Keycloak.


2. In Authentik  make a new provider like you would for most OIDC providers, noting the client ID and secret to be used later in Listmonk's settings.
    - However, be particularly careful about the following settings:
        - Copy the Redirect URI from the OIDC settings in Listmonk, described in the next section. It should not have a trailing slash.
        - For Signing Key, set it to "authentik Self-signed Certificate." Do not leave it blank, or the response is signed with an algorithm that Listmonk doesn't support.
    - When you create the Authentik application, associate it to the provider as usual. Note the slug will be used in the redirect URI. You can use `listmonk` as a slug.

3. Once the client is created, copy the Client Secret.
4. Use the Client Secret and Client ID to populate Listmonk settings.
5. Hit save
6. Create a user in listmonk with the same email as the one in your SSO
7. Try to login as this user


!!! info "Tracking issue"
    In case you need help with setting up Authentik, please [see this issue](https://github.com/knadh/listmonk/issues/2205)

### Other SSO

Feel free to contribute to this documentation if you setup another SSO provider for Listmonk.