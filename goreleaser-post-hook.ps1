#!/usr/bin/env pwsh

Write-Output "moving bazel outputs to goreleaser dist directory for packaging..."

if (-Not (Test-Path "dist")) {
  New-Item -Force -Path (Get-Location).Path -Name "dist" -ItemType "directory"
}

Copy-Item (Resolve-Path "bazel-bin/bdist/*") -Destination (Resolve-Path "dist") -Recurse -Force
