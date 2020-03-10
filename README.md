![listmonk](https://user-images.githubusercontent.com/547147/60170989-41681f00-9827-11e9-93a8-a871a40be913.png)

> listmonk is **alpha** software and may change and break. Use with caution. That said, it has been in active use at [zerodha.com](https://zerodha.com) for several months where it has processed hundreds of campaigns and tens of millions of e-mails.

listmonk is a standalone, self-hosted, newsletter and mailing list manager. It is fast, feature-rich, and packed into a single binary. It uses a PostgreSQL database as its data store.

[![listmonk-splash](https://user-images.githubusercontent.com/547147/60884802-8189c180-a26b-11e9-85ee-622e5dee8869.png)](https://listmonk.app)

### Installation and use

- Download the [latest release](https://github.com/knadh/listmonk/releases) for your platform and extract the listmonk binary. For example: `tar -C $HOME/listmonk -xzf listmonk_$VERSION_$OS_$ARCH.tar.gz`
- Navigate to the directory containing the binary (`cd $HOME/listmonk`) and run `./listmonk --new-config` to generate a sample `config.toml` and add your configuration (SMTP and Postgres DB credentials primarily).
- `./listmonk --install` to setup the DB.
- Run `./listmonk` and visit `http://localhost:9000`.
- Since there is no user auth yet, it's best to put listmonk behind a proxy like Nginx and setup basicauth on all endpoints except for the few endpoints that need to be public. Here is a [sample nginx config](https://github.com/knadh/listmonk/wiki/Production-Nginx-config) for production use.

### Running on Docker

You can pull the official Docker Image from [Docker Hub](https://hub.docker.com/r/listmonk/listmonk).

You can checkout the [docker-compose.yml](docker-compose.yml) to get an idea of how to run `listmonk` with `PostgreSQL` together using Docker.

- `docker-compose up -d app db` to run all the services together.
- `docker-compose run --rm app ./listmonk --install` to setup the DB.
- Visit `http://localhost:9000`.

Alternatively, to run a demo of listmonk, you can quickly spin up a container `docker-compose up -d demo-db demo-app`. NOTE: This doesn't persist Postgres data after you stop and remove the container, this setup is intended only for demo. _DO NOT_ use the demo setup in production.

# Configuration

## Environment variables

The Listmonk instance can be customized by specifying environment variables on the first run. The following environment values are provided to customize Listmonk:

(Note: `LISTMONK_` is the prefix and dots are replaced by double underscore `__`)

- `LISTMONK_address`: Interface and port where the app will run its webserver. Default: **"0.0.0.0:9000"**
- `LISTMONK_root`: Public root URL of the listmonk installation that'll be used in the messages for linking to images, unsubscribe page etc. Default: **"https://listmonk.mysite.com"**
- `LISTMONK_logo_url`: (Optional) full URL to the static logo to be displayed on user facing view such as the unsubscription page. eg: https://mysite.com/images/logo.svg. Default: **"https://listmonk.mysite.com/public/static/logo.png"**
- `LISTMONK_favicon_url`: (Optional) full URL to the static favicon to be displayed on user facing view such as the unsubscription page. eg: https://mysite.com/images/favicon.png. Default: **"https://listmonk.mysite.com/public/static/favicon.png"**
- `LISTMONK_from_email`: The default 'from' e-mail for outgoing e-mail campaigns. Default: **"listmonk <from@mail.com>"**
- `LISTMONK_notify_emails`: List of e-mail addresses to which admin notifications such as import updates, campaign completion, failure etc. should be sent. To disable notifications, set an empty list, eg: notify_emails = []. Default: **["admin1@mysite.com", "admin2@mysite.com"]**
- `LISTMONK_concurrency`: Maximum concurrent workers that will attempt to send messages simultaneously. This should depend on the number of CPUs the machine has and also the number of simultaenous e-mails the mail server will. Default: **"100"**
- `LISTMONK_max_send_errors`: The number of errors (eg: SMTP timeouts while e-mailing) a running campaign should tolerate before it is paused for manual investigation or intervention. Set to 0 to never pause. Default: **"1000"**
- `LISTMONK_allow_blacklist`: Allow subscribers to unsubscribe from all mailing lists and mark themselves as blacklisted? Default: **"false"**
- `LISTMONK_allow_export`: Allow subscribers to export data recorded on them? Default: **"false"**
- `LISTMONK_exportable`: Items to include in the data export. [profile] Subscriber's profile including custom attributes [subscriptions] Subscriber's subscription lists (private list names are masked) [campaign_views] Campaigns the subscriber has viewed and the view counts [link_clicks] Links that the subscriber has clicked and the click counts. Default: **["profile", "subscriptions", "campaign_views", "link_clicks"]**
- `LISTMONK_allow_wipe`: Allow subscribers to delete themselves from the database? This deletes the subscriber and all their subscriptions. Their association to campaign views and link clicks are also removed while views and click counts remain (with no subscriber associated to them) so that stats and analytics aren't affected. Default: **"false"**

- `LISTMONK_db__host`: Allows you to set the Postgres host path. Default: **"demo-db"**
- `LISTMONK_db__port`: Set the port for the Postgres connection. Default: **"5432"**
- `LISTMONK_db__user`: Set the username for the Postgres connection. Default: **"listmonk"**
- `LISTMONK_db__password`: Set the password for the Postgres connection. Default: **"listmonk"**
- `LISTMONK_db__database`: Set the database name for the Postgres connection. Default: **"listmonk"**
- `LISTMONK_db__ssl_mode`: Set the SSL mode connection. Default: **"disable"**
- `LISTMONK_db__max_open`: Maximum active connections to pool. Default: **"50"**
- `LISTMONK_db__max_idle`: Maximum idle connections to pool. Default: **"10"**

### Specifying Multiple SMTP servers allows listmonk to send emails using multiple SMTP servers for increasing throughput. ###

- `LISTMONK_smtp__my0__enable`: Enable or disable this SMTP account. Default: **"true"**
- `LISTMONK_smtp__my0__host`: Allows you to set the host for the SMTP gateway. Default: **"my.smtp.server"**
- `LISTMONK_smtp__my0__port`: Allows you to set the port for the SMTP gateway. Default: **"25"**
- `LISTMONK_smtp__my0__auth_protocol`: Authentication type (cram | plain | empty for no auth). Default: **"cram"**
- `LISTMONK_smtp__my0__username`: Allows you to set the username for the SMTP gateway. Default: **"xxxx"**
- `LISTMONK_smtp__my0__password`: Set the password for the SMTP gateway. Default: **""**
- `LISTMONK_smtp__my0__hello_hostname`: Optional. Some SMTP servers require a FQDN in the hostname. By default, HELLOs go with "localhost". Set this if a custom hostname should be used. Default: **""**
- `LISTMONK_smtp__my0__send_timeout`: Maximum time (milliseconds) to wait per e-mail push. Default: **"5000"**
- `LISTMONK_smtp__my0__max_conns`: Maximum concurrent connections to the SMTP server. Default: **"10"**

- `LISTMONK_smtp__postal__enable`: Enable or disable this SMTP account. Default: **"false"**
- `LISTMONK_smtp__postal__host`: Allows you to set the host for the SMTP gateway. Default: **"my.smtp.server2"**
- `LISTMONK_smtp__postal__port`: Allows you to set the port for the SMTP gateway. Default: **"25"**
- `LISTMONK_smtp__postal__auth_protocol`: Authentication type (cram | plain | empty for no auth). Default: **"plain"**
- `LISTMONK_smtp__postal__username`: Allows you to set the username for the SMTP gateway. Default: **"xxxx"**
- `LISTMONK_smtp__postal__password`: Set the password for the SMTP gateway. Default: **""**
- `LISTMONK_smtp__postal__hello_hostname`: Optional. Some SMTP servers require a FQDN in the hostname. By default, HELLOs go with "localhost". Set this if a custom hostname should be used. Default: **""**
- `LISTMONK_smtp__postal__send_timeout`: Maximum time (milliseconds) to wait per e-mail push. Default: **"5000"**
- `LISTMONK_smtp__postal__max_conns`: Maximum concurrent connections to the SMTP server. Default: **"10"**

### Upload settings ###

- `LISTMONK_upload__provider`: Provider which will be used to host uploaded media. Bundled providers are "filesystem" and "s3". Default: **"filesystem"**

### S3 Provider settings ###

- `LISTMONK_upload__s3__aws_access_key_id`: (Optional). AWS Access Key and Secret Key for the user to access the bucket. Leaving it empty would default to use instance IAM role. Default: **""**
- `LISTMONK_upload__s3__aws_secret_access_key`: Allows you to set the s3 AWS seceret access key. Default: **""**

- `LISTMONK_upload__s3__aws_default_region`: AWS Region where S3 bucket is hosted. Default: **"ap-south-1"**
- `LISTMONK_upload__s3__bucket`: Specify bucket name. Default: **""**
- `LISTMONK_upload__s3__bucket_path`: Path where the files will be stored inside bucket. Empty value ("") means the root of bucket. Default: **""**
- `LISTMONK_upload__s3__bucket_type`: Bucket type can be "private" or "public". Default: **"public"**
- `LISTMONK_upload__s3__expiry`: (Optional) Specify TTL (in seconds) for the generated presigned URL. Expiry value is used only if the bucket is private. Default: **"86400"**

### Filesystem provider settings ###

- `LISTMONK_upload__filesystem__upload_path`: Path to the uploads directory where media will be uploaded. Leaving it empty ("") means current working directory. Default: **""**
- `LISTMONK_upload__filesystem__upload_uri`: Upload URI that's visible to the outside world. The media uploaded to upload_path will be made available publicly under this URI, for instance, list.yoursite.com/uploads.  Default: **"/uploads"**

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
