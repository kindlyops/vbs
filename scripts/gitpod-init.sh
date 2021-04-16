#! /usr/bin/env bash

if [[ -n "${GITPOD_WORKSPACE_URL}" ]]; then
    echo "startup --output_base=/workspace/bazel_output_base" >> .bazelrc.user
fi