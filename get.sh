#!/usr/bin/env bash

set -e

RELEASES_URL="https://github.com/flyin/rancher-deploy/releases"
TAR_FILE="/tmp/rancher-deploy.tar.gz"
test -z "$TMPDIR" && TMPDIR="$(mktemp -d)"

last_version() {
  curl -sL -o /dev/null -w %{url_effective} "$RELEASES_URL/latest" |
    rev |
    cut -f1 -d'/'|
    rev
}

download() {
  test -z "$VERSION" && VERSION="$(last_version)"

  test -z "$VERSION" && {
    echo "Unable to get rancher-deploy version." >&2
    exit 1
  }

  rm -f "$TAR_FILE"

  curl -s -L -o "$TAR_FILE" "$RELEASES_URL/download/$VERSION/rancher-deploy_$(uname -s)_$(uname -m).tar.gz"
}

download
tar -xf "$TAR_FILE" -C "$TMPDIR"
"${TMPDIR}/rancher-deploy" "$@"
