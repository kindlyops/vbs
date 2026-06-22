#!/usr/bin/env pwsh

# goreleaser compiles the placeholder dummy.go (see .goreleaser.yml). This hook
# replaces one target's placeholder binary in goreleaser's dist directory with
# the real binary that bazel built under bazel-bin/bdist, before goreleaser
# archives it. goreleaser runs the post hook once per target in parallel and
# passes that target's os and arch, so each invocation swaps only its own
# binary (copying every binary on every invocation would race).
#
# goreleaser v2 appends a CPU microarchitecture level to each output directory
# (e.g. vbs_darwin_arm64_v8.0, vbs_linux_amd64_v1), while bazel emits plain
# vbs_<os>_<arch> directories. Match by os/arch prefix so the swap keeps working
# regardless of which microarchitecture suffix goreleaser chooses.

param(
    [Parameter(Mandatory = $true)][string]$Os,
    [Parameter(Mandatory = $true)][string]$Arch
)

$ErrorActionPreference = "Stop"

$name = "vbs_${Os}_${Arch}"
$srcDir = Join-Path (Resolve-Path "bazel-bin/bdist") $name
if (-Not (Test-Path $srcDir)) {
    throw "no bazel output at $srcDir for target $Os/$Arch"
}

$destDirs = Get-ChildItem -Path "dist" -Directory -Filter "$name*"
if ($destDirs.Count -eq 0) {
    throw "no goreleaser dist directory matches $name; cannot swap in the bazel binary"
}

foreach ($destDir in $destDirs) {
    Copy-Item -Path (Join-Path $srcDir "*") -Destination $destDir.FullName -Recurse -Force
    Write-Output "swapped bazel binary $name -> dist/$($destDir.Name)"
}
