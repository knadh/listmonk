# listmonk

Full fledged, high performance newsletter and mailing list manager

# Development

Install dependencies

```bash
make deps
```

Build frontend assets

```bash
make build_frontend
```

Create config file and edit the necessary params

```bash
cp config.toml.sample config.toml
```

Use [stuffbin](https://github.com/knadh/stuffbin) to package static assets and build binary

```bash
make build
```

Binary comes up with installer to setup schema and superadmin

```bash
./listmonk --install
```

Run binary
```bash
./listmonk
```

For new developers, you can also run all at once using `quickdev` option.

```bash
make quickdev
```
