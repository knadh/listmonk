# Dev

There are two independent components, the Go backend and the React frontend. In the dev environment, both have to run.

### First time setup

1. Write `config.toml`
2. Run `listmonk --install` to do the DB setup.

# Dev environment

### 1. Run the backend

`make deps` (once, to install dependencies) and then `make run`

### 2. Run the frontend

`cd frontend/my` and `yarn start`

### 3. Setup an Nginx proxy endpoint

```
# listmonk
server {
    listen 9001;

    # Proxy all /api/* requests to the Go backend.
    location /api {
        proxy_pass http://localhost:9000;
    }

    # Proxy everything else to the yarn server
    # that's running the React frontend.
    location / {
        proxy_pass http://localhost:3000;
    }
}

```

Visit `http://localhost:9001` to access the frontend running in development mode where Javascript updates are pushed live.

# Production

1. To build the Javascript frontend, run `make build-frontend`. This only needs to be run if the frontend has changed.

2. `make build` builds a single production ready binary with all static assets embeded in it. Make sure to have installed the dependencies with `make deps` once.

---

# TODO: Essentials for v0.10

- update list time after import
- dockerize
- add an http call to do version checks and alerts
- make the design responsive
- error pause should be % and not absolute
- views for faster dashboard analytics
- bounce processing
- docs
- tests

### Features

- running campaigns widget on the dashboard
- analytics
- GDPR

# Features

Features

- userdb
- campaign error logs
- upgrade + migration
- views for fast analytics widgets

- tab navigation in links and buttons
- inject semver into build version
- permalink for lists
- props.config race condition
- app.Messenger is flawed. Add multi-messenger support to app as well.
