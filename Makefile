# Try to get the commit hash from 1) git 2) the VERSION file 3) fallback.
LAST_COMMIT := $(or $(shell git rev-parse --short HEAD 2> /dev/null),$(shell head -n 1 VERSION | grep -oP -m 1 "^[a-z0-9]+$$"),"")

# Try to get the semver from 1) git 2) the VERSION file 3) fallback.
VERSION := $(or $(LISTMONK_VERSION),$(shell git describe --tags --abbrev=0 2> /dev/null),$(shell grep -oP 'tag: \Kv\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?' VERSION),"v0.0.0")

BUILDDATE := $(if $(SOURCE_DATE_EPOCH),$(shell date -u -d @$(SOURCE_DATE_EPOCH) +"%Y-%m-%dT%H:%M:%S%z"),$(shell date -u +"%Y-%m-%dT%H:%M:%S%z"))
BUILDSTR := ${VERSION} (\#${LAST_COMMIT} $(BUILDDATE))

GOPATH ?= $(HOME)/go
STUFFBIN ?= $(GOPATH)/bin/stuffbin

# SSR admin frontend (built from static/admin/assets -> static/admin/dist).
FRONTEND = static/admin
FRONTEND_DIST = $(FRONTEND)/dist
FRONTEND_NODE_MODULES = $(FRONTEND)/node_modules
FRONTEND_DEPS_STAMP = $(FRONTEND_NODE_MODULES)/.installed

# Feather icons SVG sprite generated from icons.txt.
FRONTEND_ICONS_LIST = $(FRONTEND)/icons.txt
FRONTEND_ICONS = $(FRONTEND)/assets/static/icons.svg

FRONTEND_DEPS = \
	$(FRONTEND_DEPS_STAMP) \
	$(FRONTEND)/package.json \
	$(FRONTEND)/bun.lock \
	$(FRONTEND)/build.mjs \
	$(shell find $(FRONTEND)/assets -type f)

BIN := listmonk
STATIC := config.toml.sample \
	schema.sql queries:/queries permissions.json \
	static/public:/public \
	static/admin/views:/admin/views \
	static/admin/partials:/admin/partials \
	static/admin/dist:/admin/static \
	static/admin/i18n.txt:/admin/i18n.txt \
	static/email-templates \
	i18n:/i18n

SQL := $(shell find . -type f -name "*.sql") $(shell find queries -type f -name "*.sql")
SRC := $(shell find . -type f -name "*.go")

.PHONY: build
build: $(BIN)

$(STUFFBIN):
	go install github.com/knadh/stuffbin/...

# Build the backend to ./listmonk.
$(BIN): $(SRC) go.mod go.sum schema.sql $(SQL) permissions.json
	CGO_ENABLED=0 go build -o ${BIN} -ldflags="-s -w -X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}'" ./cmd

# Run the backend in dev mode. DEV_CONFIG is set by the dev container to use
# its config and initialize a fresh DB.
GO_RUN = CGO_ENABLED=0 go run -ldflags="-s -w -X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}'" ./cmd

.PHONY: run
run: $(FRONTEND_DIST)
ifneq ($(DEV_CONFIG),)
	$(GO_RUN) --config="$(DEV_CONFIG)" --install --idempotent --yes
endif
	$(GO_RUN) $(if $(DEV_CONFIG),--config="$(DEV_CONFIG)")

# Install SSR admin frontend deps.
$(FRONTEND_DEPS_STAMP): $(FRONTEND)/package.json $(FRONTEND)/bun.lock
	cd $(FRONTEND) && bun install --frozen-lockfile
	touch $@

# Generate svg icon sprite.
$(FRONTEND_ICONS): $(FRONTEND_ICONS_LIST) scripts/build-icons.py
	python3 scripts/build-icons.py --icons $(FRONTEND_ICONS_LIST) --out $(FRONTEND_ICONS)

.PHONY: build-icons
build-icons: $(FRONTEND_ICONS)

# Build the SSR admin frontend (Bun) into static/admin/dist.
$(FRONTEND_DIST): $(FRONTEND_ICONS) $(FRONTEND_DEPS)
	cd $(FRONTEND) && bun run build
	touch -c $(FRONTEND_DIST)

.PHONY: build-frontend
build-frontend: $(FRONTEND_DIST)

# Run Go tests.
.PHONY: test
test:
	go test ./...

# Bundle all static assets including the SSR admin frontend into the ./listmonk binary
# using stuffbin.
.PHONY: dist
dist: $(STUFFBIN) build build-frontend pack-bin

# pack-bin runs stuffbin packing on the given binary. This is used
# in the .goreleaser post-build hook.
.PHONY: pack-bin
pack-bin: build-frontend $(BIN) $(STUFFBIN)
	$(STUFFBIN) -a stuff -in ${BIN} -out ${BIN} ${STATIC}

# Use goreleaser to do a dry run producing local builds.
.PHONY: release-dry
release-dry:
	goreleaser release --parallelism 1 --clean --snapshot --skip=publish

# Use goreleaser to build production releases and publish them.
.PHONY: release
release:
	goreleaser release --parallelism 1 --clean

# Build the dev environmentt.
.PHONY: build-dev-docker
build-dev-docker:
	docker compose -f dev/docker-compose.yml build

# Start the dev services and open a shell.
.PHONY: run-dev-docker
run-dev-docker:
	docker compose -f dev/docker-compose.yml up -d --wait
	docker compose -f dev/docker-compose.yml exec dev bash

# Stop the dev services (db is preserved).
.PHONY: stop-dev-docker
stop-dev-docker:
	docker compose -f dev/docker-compose.yml down

# Remove the the dev environment and storage.
.PHONY: rm-dev-docker
rm-dev-docker:
	docker compose -f dev/docker-compose.yml down -v
