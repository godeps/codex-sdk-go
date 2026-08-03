# Release Verification

Use this checklist before tagging or publishing a release candidate.

## Documentation And API Shape

Run these first when you touched docs or public APIs:

```bash
GOWORK=off go doc -all .
GOWORK=off go test ./example/...
```

## Core Test Matrix

```bash
GOWORK=off go test ./cmd/codex-sdk-gen
GOWORK=off go test ./cmd/codex-sdk-runtime
GOWORK=off go test ./internal/runtimebin ./internal/router
GOWORK=off go test ./...
GOWORK=off go test -race ./...
```

`go test ./...` is the broadest regression check. `go test -race ./...` is the stronger concurrency
check and should be run before a release candidate is cut.

## Runtime And Packaging Checks

```bash
GOWORK=off go run ./cmd/codex-sdk-runtime version 0.144.4
GOWORK=off go run ./cmd/codex-sdk-gen verify
```

If you changed runtime packaging, also run the offline install and path-resolution commands from
[runtime-installation.md](runtime-installation.md).

## Evidence Notes

- Capture the command output you used for the release decision.
- If a command fails because a pinned external checkout is missing, record the blocker instead of
  treating the check as passed.
- Do not mark compatibility ledger rows or coverage rows as `verified` without the matching test
  evidence.
