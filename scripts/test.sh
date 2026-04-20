#!/usr/bin/env bash
# Run unit tests with race detection and coverage.
set -euo pipefail

cd "$(dirname "$0")/.."

go test -race -cover ./...
