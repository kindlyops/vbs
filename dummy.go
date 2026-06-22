//go:build neverbuild
// +build neverbuild

// This file is the placeholder main that goreleaser compiles (see
// .goreleaser.yml `main: dummy.go`). The real binary is built by bazel and
// swapped into goreleaser's dist directory by goreleaser-post-hook.ps1. The
// neverbuild constraint keeps `go build ./...`, `go test`, and bazel from
// picking up this empty main; goreleaser bypasses the constraint because it
// names the file explicitly on the `go build` command line.
//
// If you are seeing this message at runtime, the bazel binary was NOT swapped
// in during release and the placeholder shipped instead. That is a packaging
// bug, not a usable build of vbs.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr,
		"vbs: this is the placeholder build; the real binary was not packaged "+
			"during release. Please report this at "+
			"https://github.com/kindlyops/vbs/issues")
	os.Exit(1)
}
