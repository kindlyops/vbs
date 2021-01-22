#!/usr/bin/env bash
set -eo pipefail

echo "moving bazel outputs to goreleaser dist directory for packaging..."
mkdir -p dist/vbs_darwin_amd64
mkdir -p dist/vbs_linux_amd64
mkdir -p dist/vbs_windows_amd64

cp bazel-bin/bdist/vbs_darwin_amd64/vbs dist/vbs_darwin_amd64/vbs
cp bazel-bin/bdist/vbs_linux_amd64/vbs dist/vbs_linux_amd64/vbs
cp bazel-bin/bdist/vbs_windows_amd64/vbs.exe dist/vbs_windows_amd64/vbs.exe

