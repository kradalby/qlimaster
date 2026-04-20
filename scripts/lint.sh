#!/usr/bin/env bash
# Run golangci-lint locally.
set -euo pipefail

cd "$(dirname "$0")/.."

golangci-lint run --timeout=5m ./...
