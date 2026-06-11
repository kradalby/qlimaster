#!/usr/bin/env bash
# Watch the most recent CI run on the current branch until it completes.
set -euo pipefail

branch="$(git branch --show-current)"
run_id="$(gh run list --workflow=ci.yml --branch="$branch" --limit=1 \
	--json databaseId --jq '.[0].databaseId')"

if [ -z "$run_id" ]; then
	echo "no CI runs found on branch $branch"
	exit 1
fi

gh run watch "$run_id"
