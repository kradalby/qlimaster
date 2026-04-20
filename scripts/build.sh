#!/usr/bin/env bash
# Build the qlimaster binary into ./qlimaster.
set -euo pipefail

cd "$(dirname "$0")/.."

version="${QLIMASTER_VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"

CGO_ENABLED=0 go build \
	-trimpath \
	-ldflags "-s -w -X main.version=${version}" \
	-o qlimaster \
	./cmd/qlimaster

echo "built ./qlimaster (version ${version})"
