# Try to get the commit hash from 1) git 2) the VERSION file 3) fallback.
LAST_COMMIT := $(or $(shell git rev-parse --short HEAD 2> /dev/null),$(shell head -n 1 VERSION | grep -oP -m 1 "^[a-z0-9]+$$"),"")

# Try to get the semver from 1) git 2) the VERSION file 3) fallback.
VERSION := $(or $(LISTMONK_VERSION),$(shell git describe --tags --abbrev=0 2> /dev/null),$(shell grep -oP 'tag: \Kv\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?' VERSION),"v0.0.0")

BUILDDATE := $(if $(SOURCE_DATE_EPOCH),$(shell date -u -d @$(SOURCE_DATE_EPOCH) +"%Y-%m-%dT%H:%M:%S%z"),$(shell date -u +"%Y-%m-%dT%H:%M:%S%z"))
BUILDSTR := ${VERSION} (\#${LAST_COMMIT} $(BUILDDATE))

GOPATH ?= $(HOME)/go
STUFFBIN ?= $(GOPATH)/bin/stuffbin

# SSR admin frontend (built from static/admin/src -> static/admin/dist).
FRONTEND = static/admin
FRONTEND_DIST = $(FRONTEND)/dist
FRONTEND_NODE_MODULES = $(FRONTEND)/node_modules
FRONTEND_DEPS = \
	$(FRONTEND_NODE_MODULES) \
	$(FRONTEND)/package.json \
	$(FRONTEND)/build.mjs \
	$(shell find $(FRONTEND)/src -type f)

BIN := listmonk
STATIC := config.toml.sample \
	schema.sql queries:/queries permissions.json \
	static/public:/public \
	static/admin/views:/admin/views \
	static/admin/partials:/admin/partials \
	static/admin/dist:/admin/static \
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

# Run the backend in dev mode. The SSR admin assets are loaded from disk from
# static/admin/dist, so build them first.
.PHONY: run
run: $(FRONTEND_DIST)
	CGO_ENABLED=0 go run -ldflags="-s -w -X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}'" ./cmd

# Install SSR admin frontend deps.
$(FRONTEND_NODE_MODULES): $(FRONTEND)/package.json
	cd $(FRONTEND) && bun install
	touch -c $(FRONTEND_NODE_MODULES)

# Build the SSR admin frontend (Bun) into static/admin/dist.
$(FRONTEND_DIST): $(FRONTEND_DEPS)
	cd $(FRONTEND) && bun run build
	touch -c $(FRONTEND_DIST)

.PHONY: build-frontend
build-frontend: $(FRONTEND_DIST)

# Run Go tests.
.PHONY: test
test:
	go test ./...

# Bundle all static assets including the JS frontends into the ./listmonk binary
# using stuffbin (installed with make deps).
.PHONY: dist
dist: $(STUFFBIN) build build-frontend pack-bin

# pack-releases runns stuffbin packing on the given binary. This is used
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

# Build local docker images for development.
.PHONY: build-dev-docker
build-dev-docker: build ## Build docker containers for the entire suite (Front/Core/PG).
	cd dev; \
	docker compose build ; \

# Spin a local docker suite for local development.
.PHONY: dev-docker
dev-docker: build-dev-docker ## Build and spawns docker containers for the entire suite (Front/Core/PG).
	cd dev; \
	docker compose up

# Run the backend in docker-dev mode. The SSR admin assets are loaded from disk from static/admin/dist.
.PHONY: run-backend-docker
run-backend-docker:
	CGO_ENABLED=0 go run -ldflags="-s -w -X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}'" ./cmd --config=dev/config.toml

# Tear down the complete local development docker suite.
.PHONY: rm-dev-docker
rm-dev-docker: build ## Delete the docker containers including DB volumes.
	cd dev; \
	docker compose down -v ; \

# Setup the db for local dev docker suite.
.PHONY: init-dev-docker
init-dev-docker: build-dev-docker ## Delete the docker containers including DB volumes.
	cd dev; \
	docker compose run --rm backend sh -c "make dist && ./listmonk --install --idempotent --yes --config dev/config.toml"
