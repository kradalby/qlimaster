#!/usr/bin/env bash
# Run a short fuzz session against the score parser.
set -euo pipefail

cd "$(dirname "$0")/.."

duration="${1:-30s}"

go test -fuzz=. -fuzztime="$duration" ./score
