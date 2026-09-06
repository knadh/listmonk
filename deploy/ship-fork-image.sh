#!/usr/bin/env bash
# Build the listmonk fork into a self-contained linux/amd64 image and ship it to the
# server over SSH (docker save | docker load), no registry. KISS deploy for the fork.
#
# Usage:
#   DEPLOY_HOST=1.2.3.4 ./deploy/ship-fork-image.sh            # build + ship
#   BUILD_ONLY=1 ./deploy/ship-fork-image.sh                   # build locally only
#   DEPLOY_HOST=1.2.3.4 RUN_DEPLOY=1 ./deploy/ship-fork-image.sh  # ship + run remote deploy.sh
#
# Requires on the build host: docker (with buildx), Go, Node + Yarn (frontend toolchain).
set -euo pipefail

IMAGE="${IMAGE:-listmonk-fork}"
PLATFORM="${PLATFORM:-linux/amd64}"
DEPLOY_USER="${DEPLOY_USER:-deploy}"
DEPLOY_HOST="${DEPLOY_HOST:-}"
REMOTE_INFRA_DIR="${REMOTE_INFRA_DIR:-/home/deploy/infra}"
BUILD_ONLY="${BUILD_ONLY:-0}"
RUN_DEPLOY="${RUN_DEPLOY:-0}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

SHA="$(git rev-parse --short HEAD)"
VERSION="${LISTMONK_VERSION:-v6.3.0-fork+$SHA}"
TAG="$IMAGE:prod"
TAG_SHA="$IMAGE:$SHA"

# corepack ships Yarn 1.x with Node; fall back to it if yarn isn't on PATH.
YARN="${YARN:-yarn}"
command -v "$YARN" >/dev/null 2>&1 || YARN="corepack yarn"

echo "==> building self-contained binary ($PLATFORM, version $VERSION)"
# stuffbin packs assets onto the binary and runs on the host, so install it for the host
# arch first; otherwise the cross-compile env below would build a linux stuffbin make can't run.
GOOS="$(go env GOHOSTOS)" GOARCH="$(go env GOHOSTARCH)" go install github.com/knadh/stuffbin/...@latest
# Drop any stale (host-arch) binary so make actually cross-compiles instead of reusing it.
rm -f listmonk
GOOS="${PLATFORM%%/*}" GOARCH="${PLATFORM##*/}" LISTMONK_VERSION="$VERSION" \
  make dist YARN="$YARN"

echo "==> building image $TAG"
docker buildx build --platform "$PLATFORM" -t "$TAG" -t "$TAG_SHA" --load .

if [[ "$BUILD_ONLY" == "1" ]]; then
  echo "==> build-only: $TAG ready locally ($VERSION)"
  exit 0
fi

[[ -n "$DEPLOY_HOST" ]] || { echo "ERROR: set DEPLOY_HOST (or BUILD_ONLY=1)" >&2; exit 1; }

echo "==> shipping $TAG to $DEPLOY_USER@$DEPLOY_HOST"
docker save "$TAG" "$TAG_SHA" | ssh "$DEPLOY_USER@$DEPLOY_HOST" 'docker load'

if [[ "$RUN_DEPLOY" == "1" ]]; then
  echo "==> running remote deploy.sh"
  ssh "$DEPLOY_USER@$DEPLOY_HOST" "bash $REMOTE_INFRA_DIR/scripts/deploy.sh"
else
  echo "==> image loaded. Deploy via your pipeline (push selfhosted-tools), or re-run with RUN_DEPLOY=1."
fi
