#!/usr/bin/env bash
set -eo pipefail

export RUNFILES_DIR="$PWD"/..
export PATH="$PWD/external/go_sdk/bin:$PATH"
gazelle="$PWD/$1"

echo "Using these commands"
command -v golangci-lint
echo "$gazelle"

cd "$BUILD_WORKSPACE_DIRECTORY"

golangci-lint run
