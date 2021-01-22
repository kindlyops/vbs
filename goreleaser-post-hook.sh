#!/usr/bin/env bash
set -eo pipefail

echo "moving bazel outputs to goreleaser dist directory for packaging..."
mkdir -p dist/vbs_darwin_amd64
mkdir -p dist/vbs_linux_amd64
mkdir -p dist/vbs_windows_amd64

sudo cp bazel-bin/bdist/vbs-darwin dist/vbs_darwin_amd64/vbs
sudo cp bazel-bin/bdist/vbs-linux dist/vbs_linux_amd64/vbs
sudo cp bazel-bin/bdist/vbs-windows.exe dist/vbs_windows_amd64/vbs.exe

