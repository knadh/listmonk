![listmonk](https://user-images.githubusercontent.com/547147/60170989-41681f00-9827-11e9-93a8-a871a40be913.png)

> listmonk is **alpha** software and may change and break. Use with caution. That said, it has been in active use at [zerodha.com](https://zerodha.com) for several months where it has processed hundreds of campaigns and tens of millions of e-mails.

listmonk is a standalone, self-hosted, newsletter and mailing list manager. It is fast, feature-rich, and packed into a single binary. It uses a PostgreSQL database as its data store.

[![listmonk-splash](https://user-images.githubusercontent.com/547147/60884802-8189c180-a26b-11e9-85ee-622e5dee8869.png)](https://listmonk.app)

### Installation and use

- Download the [latest release](https://github.com/knadh/listmonk/releases) for your platform and extract the listmonk binary. For example: `tar -C $HOME/listmonk -xzf listmonk_$VERSION_$OS_$ARCH.tar.gz`
- Navigate to the directory containing the binary (`cd $HOME/listmonk`) and run `./listmonk --new-config` to generate a sample `config.toml` and add the DB configuration.
- `./listmonk --install` to setup the DB.
- Run `./listmonk` and visit `http://localhost:9000`.
- Visit the `Settings` page to configure your instance.
- Since there is no user auth yet, it's best to put listmonk behind a proxy like Nginx and setup basicauth on all endpoints except for the few endpoints that need to be public. Here is a [sample nginx config](https://github.com/knadh/listmonk/wiki/Production-Nginx-config) for production use.

### Configuration and customization
See the [configuration Wiki page](https://github.com/knadh/listmonk/wiki/Configuration).

### Running on Docker

You can pull the official Docker Image from [Docker Hub](https://hub.docker.com/r/listmonk/listmonk).

You can checkout the [docker-compose.yml](docker-compose.yml) to get an idea of how to run `listmonk` with `PostgreSQL` together using Docker (also see [configuring with environment variables](https://github.com/knadh/listmonk/wiki/Configuration)). Please visit [INSTALL.md](INSTALL.md) for detailed instructions on how to setup Listmonk with Docker.

**Alternatively**, to run a demo of listmonk, you can quickly spin up a local setup with:

`docker-compose up -d demo-db demo-app`.

**NOTE:** This doesn't persist Postgres data after you stop and remove the container, this setup is intended only for demo. _DO NOT_ use the demo setup in production.

### Other deployments

#### Heroku 

Using the [Nginx buildpack](https://github.com/heroku/heroku-buildpack-nginx) can be used to deploy listmonk on Heroku and use Nginx as a proxy to setup basicauth. 
This one-click [Heroku deploy button](https://github.com/bumi/listmonk-heroku) provides an automated default deployment.

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/bumi/listmonk-heroku)

Please note that [configuration options](https://github.com/knadh/listmonk/wiki/Configuration) must be set using [environment configruation variables](https://devcenter.heroku.com/articles/config-vars).

### Help and docs

[Help and documentation](https://listmonk.app/docs) (work in progress).

### Current features

- Admin dashboard
- Public, private, single and double optin lists (with optin campaigns)
- Fast bulk subscriber import
- Custom subscriber attributes
- Subscriber querying and segmentation with ad-hoc SQL expressions
- Subscriber data wipe / export privacy features
- Rich programmable Go HTML templates and WYSIWYG editor
- Media gallery (disk and S3 storage)
- Multi-threaded multi-SMTP e-mail queues for fast campaign delivery
- HTTP/JSON APIs for everything
- Clicks and view tracking
- and more ...

### Todo

- DB migrations
- Bounce tracking
- User auth, management, permissions
- Ability to write raw campaign logs to a target
- Analytics views and reports
- Better widgets on dashboard
- Tests!

## Developers

listmonk is free, open source software licensed under AGPLv3. There are several essential features such as user auth/management and bounce tracking that are currently missing. Contributions are welcome.

The backend is written in Go and the frontend is in React with Ant Design for UI. See [developer setup](https://github.com/knadh/listmonk/wiki/Developer-setup) to get started.

## License

listmonk is licensed under the AGPL v3 license.
