# Try to get the commit hash from 1) git 2) the VERSION file 3) fallback.
LAST_COMMIT := $(or $(shell git rev-parse --short HEAD 2> /dev/null),$(shell head -n 1 VERSION | grep -oP -m 1 "^[a-z0-9]+$$"),"")

# Try to get the semver from 1) git 2) the VERSION file 3) fallback.
VERSION := $(or $(LISTMONK_VERSION),$(shell git describe --tags --abbrev=0 2> /dev/null),$(shell grep -oP 'tag: \Kv\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?' VERSION),"v0.0.0")

BUILDSTR := ${VERSION} (\#${LAST_COMMIT} $(shell date -u +"%Y-%m-%dT%H:%M:%S%z"))

YARN ?= yarn
GOPATH ?= $(HOME)/go
STUFFBIN ?= $(GOPATH)/bin/stuffbin
FRONTEND_YARN_MODULES = frontend/node_modules
FRONTEND_DIST = frontend/dist
FRONTEND_EMAIL_BUILDER_DIST = frontend/public/static/email-builder
FRONTEND_DEPS = \
	$(FRONTEND_YARN_MODULES) \
	$(FRONTEND_EMAIL_BUILDER_DIST) \
	frontend/index.html \
	frontend/package.json \
	frontend/vite.config.js \
	frontend/.eslintrc.js \
	$(shell find frontend/fontello frontend/public frontend/src -type f)
EMAIL_BUILDER_YARN_MODULES = email-builder/node_modules
EMAIL_BUILDER_DIST = email-builder/dist
EMAIL_BUILDER_DEPS = \
	$(EMAIL_BUILDER_YARN_MODULES) \
	email-builder/package.json \
	email-builder/tsconfig.json \
	email-builder/vite.config.ts \
	$(shell find email-builder/src -type f)

BIN := listmonk
STATIC := config.toml.sample \
	schema.sql queries.sql permissions.json \
	static/public:/public \
	static/email-templates \
	frontend/dist:/admin \
	i18n:/i18n

.PHONY: build
build: $(BIN)

$(STUFFBIN):
	go install github.com/knadh/stuffbin/...

$(FRONTEND_YARN_MODULES): frontend/package.json frontend/yarn.lock
	cd frontend && $(YARN) install
	touch -c $(FRONTEND_YARN_MODULES)

$(EMAIL_BUILDER_YARN_MODULES): frontend/package.json frontend/yarn.lock
	cd email-builder && $(YARN) install
	touch -c $(EMAIL_BUILDER_YARN_MODULES)

# Build the backend to ./listmonk.
$(BIN): $(shell find . -type f -name "*.go") go.mod go.sum schema.sql queries.sql permissions.json
	CGO_ENABLED=0 go build -o ${BIN} -ldflags="-s -w -X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}'" cmd/*.go

# Run the backend in dev mode. The frontend assets in dev mode are loaded from disk from frontend/dist.
.PHONY: run
run:
	CGO_ENABLED=0 go run -ldflags="-s -w -X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}' -X 'main.frontendDir=frontend/dist'" cmd/*.go

# Build the JS frontend into frontend/dist.
$(FRONTEND_DIST): $(FRONTEND_DEPS)
	export VUE_APP_VERSION="${VERSION}" && cd frontend && $(YARN) build
	touch -c $(FRONTEND_DIST)

# Build the JS email-builder dist.
$(EMAIL_BUILDER_DIST): $(EMAIL_BUILDER_DEPS)
	export VUE_APP_VERSION="${VERSION}" && cd email-builder && $(YARN) build
	touch -c $(EMAIL_BUILDER_DIST)

# Copy the build assets to frontend.
$(FRONTEND_EMAIL_BUILDER_DIST): $(EMAIL_BUILDER_DIST)
	mkdir -p $(FRONTEND_EMAIL_BUILDER_DIST)
	cp -r $(EMAIL_BUILDER_DIST)/* $(FRONTEND_EMAIL_BUILDER_DIST)
	touch -c $(FRONTEND_EMAIL_BUILDER_DIST)

.PHONY: build-frontend
build-frontend: $(FRONTEND_EMAIL_BUILDER_DIST) $(FRONTEND_DIST)

.PHONY: build-email-builder
build-email-builder: $(EMAIL_BUILDER_DIST) $(FRONTEND_EMAIL_BUILDER_DIST)

# Run the JS frontend server in dev mode.
.PHONY: run-frontend
run-frontend:
	export VUE_APP_VERSION="${VERSION}" && cd frontend && $(YARN) dev

# Run Go tests.
.PHONY: test
test:
	go test ./...

# Bundle all static assets including the JS frontend into the ./listmonk binary
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

# Run the backend in docker-dev mode. The frontend assets in dev mode are loaded from disk from frontend/dist.
.PHONY: run-backend-docker
run-backend-docker:
	CGO_ENABLED=0 go run -ldflags="-s -w -X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}' -X 'main.frontendDir=frontend/dist'" cmd/*.go --config=dev/config.toml

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
