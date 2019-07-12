![listmonk](https://user-images.githubusercontent.com/547147/60170989-41681f00-9827-11e9-93a8-a871a40be913.png)

> listmonk is **alpha** software and may change and break. Use with caution. That said, it has been in active use at [zerodha.com](https://zerodha.com) for several months where it has processed hundreds of campaigns and tens of millions of e-mails.

listmonk is a standalone, self-hosted, newsletter and mailing list manager. It is fast, feature-rich, and packed into a single binary. It uses a PostgreSQL database as its data store.

[![listmonk-splash](https://user-images.githubusercontent.com/547147/60884802-8189c180-a26b-11e9-85ee-622e5dee8869.png)](https://listmonk.app)

### Installation and use

- Download the [latest release](https://github.com/knadh/listmonk/releases) and extract the listmonk binary somewhere.
- Run `./listmonk --new-config` to generate a sample `config.toml` and add your configuration (SMTP and Postgres DB credentials primarily).
- `./listmonk --install` to setup the DB.
- Visit `http://localhost:9000`.
- Since there is no user auth yet, it's best to put listmonk behind a proxy like Nginx and setup basicauth on all endpoints except for the few endpoints that need to be public. Here is a [sample nginx config](https://github.com/knadh/listmonk/wiki/Production-Nginx-config) for production use.

### Running on Docker

You can checkout the [docker-compose.yml](docker-compose.yml) to get an idea of how to run `listmonk` with `PostgreSQL` together using Docker.

- `docker-compose up -d` to run all the services together.
- `docker-compose run --rm app ./listmonk --install` to setup the DB.
- Visit `http://localhost:9000`.

Alternatively, to run a demo of listmonk, you can quickly spin up a container `docker-compose up -d demo-db demo-app`. NOTE: This doesn't persist Postgres data after you stop and remove the container, this setup is intended only for demo. _DO NOT_ use the demo setup in production.

### Help and docs

[Help and documentation](https://listmonk.app/docs) (work in progress).

### Current features

- Admin dashboard
- Multiple public and private lists
- Fast bulk subscriber import
- Custom subscriber attributes
- Subscriber querying and segmentation with ad-hoc SQL
- Rich Go templates and WYSIWYG editor
- Media gallery
- Multi-threaded multi-SMTP e-mail queues for fast campaign delivery
- HTTP/JSON APIs for everything
- Clicks and view tracking
- and more ...

### Todo

- DB migrations
- Bounce tracking
- User auth, management, permissions
- Privacy features for subscribers (Download and wipe all tracking data)
- Ability to write raw campaign logs to a target
- Analytics views and reports
- Make Ant design UI components responsive
- Better widgets on dashboard
- Tests!

## Developers

listmonk is free, open source software licensed under AGPLv3. There are several essential features such as user auth/management and bounce tracking that are currently missing. Contributions are welcome.

The backend is written in Go and the frontend is in React with Ant Design for UI. See [developer setup](https://github.com/knadh/listmonk/wiki/Developer-setup) to get started.

## License

listmonk is licensed under the AGPL v3 license.
