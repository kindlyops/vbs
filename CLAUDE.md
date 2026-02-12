# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VBS (Video Broadcasting Stuff) is a Go CLI tool for video broadcast production. It contains several integrated tools: an IVS-OSC bridge for AWS IVS metadata, a lighting bridge for Companion StreamDeck control, media chapter utilities (ffmpeg/ffprobe), a fullscreen mpv video player, and a PocketBase backend server with embedded Next.js web UI.

## Build & Test Commands

The primary build system is **Bazel**. Dependencies are vendored.

```bash
bazel build //...              # Build everything
bazel test //...               # Run all tests
bazel coverage //...           # Tests with coverage (used in CI)
bazel run vbs                  # Build and run the CLI
bazel run //:vendor            # Sync/vendor Go dependencies (runs go mod tidy + go mod vendor + gazelle)
bazel run //:lint              # Run golangci-lint
```

Go commands also work for quick iteration on Go code:

```bash
go build ./...                 # Build with Go directly
go test ./cmd/ -run TestName -v  # Run a single test
go test ./...                  # Run all Go tests
```

## Linting

Configured via `.golangci.yml` with golangci-lint v2. Key constraints:
- Max function length: 100 lines / 50 statements
- Max cyclomatic complexity: 10
- Max line length: 120 characters
- Formatters: gci, gofmt, goimports

CI runs golangci-lint and shellcheck on PRs via reviewdog.

## Architecture

All CLI commands live in `cmd/` and register themselves via `init()` functions onto a root Cobra-like command (using the `muesli/coral` fork). Entry point is `main.go`.

**Platform-specific IPC**: The mpv player control uses Unix domain sockets on Linux/macOS (`cmd/play_unix.go`) and Windows named pipes (`cmd/play_windows.go`). Both export `GetIPCName()` and `ConnectIPC()`.

**Embedded web UI**: Next.js static assets are built by Bazel (see `glue.bzl`), embedded via Go `embed` in `embeddy/`, and served by the PocketBase server (`cmd/fly.go`).

**Key frameworks**:
- `charmbracelet/bubbletea` + `bubbles` + `lipgloss` for the player TUI
- `labstack/echo` for HTTP routing
- `pocketbase` for the backend server
- `rs/zerolog` for structured logging
- `spf13/viper` for configuration

## Pull Requests

When creating pull requests, always add appropriate labels to help categorize and track the changes. Valid labels: `bug`, `feature`, `security`, `maintenance`, `documentation`, `dependencies`.

When creating a PR, ask if it should be set to auto-merge.
