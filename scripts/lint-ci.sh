#!/usr/bin/env bash
# Show the latest CI lint job result on the current branch via gh.
set -euo pipefail

branch="$(git branch --show-current)"
run_id="$(gh run list --workflow=ci.yml --branch="$branch" --limit=1 \
	--json databaseId --jq '.[0].databaseId')"

if [ -z "$run_id" ]; then
	echo "no CI runs found on branch $branch"
	exit 1
fi

echo "latest CI run on $branch: $run_id"
gh run view "$run_id" --json status,conclusion,jobs \
	--jq '.status, .conclusion, (.jobs[] | "  " + .name + ": " + .conclusion)'

echo
echo "--- failed step logs (if any) ---"
gh run view "$run_id" --log-failed || true
