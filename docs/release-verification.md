# Release Verification

Use this checklist before dispatching the release workflow for a release candidate.

## Evidence-Gated Tag Creation

`v0.3.1` and later releases are evidence-gated. Do not push the release tag first and do not use a
tag push as the release trigger.

1. Pick the target commit SHA and confirm all required workflows succeeded on that exact commit:
   `ci`, `stress`, `runtime-tests`, and `runtime-native`.
2. Collect the successful run IDs for those workflows and the runtime version pinned in
   `reference.lock.json`.
3. Dispatch `.github/workflows/release.yml` with:
   - `target_commit=<40-char sha>`
   - `tag=v0.3.1` (or the candidate you are publishing)
   - `runtime_version=<reference.lock.json runtimeVersion>`
   - the four successful workflow run IDs
   - `create_tag_if_missing=true` only after the evidence bundle inputs are ready
4. Let the workflow validate the run IDs, download artifacts, build `release-bundle.json`, and only
   then create and push the tag if it is still missing.

The `ci` workflow ignores `v*` tag pushes, so tag creation inside `release.yml` does not start a
duplicate CI run.

## Documentation And API Shape

Run these first when you touched docs or public APIs:

```bash
GOWORK=off go doc -all .
GOWORK=off go test ./example/...
GOWORK=off go test
```

## Core Test Matrix

```bash
GOWORK=off go test ./cmd/codex-sdk-gen
GOWORK=off go test ./cmd/codex-sdk-runtime
GOWORK=off go test ./internal/runtimebin ./internal/router
GOWORK=off go test ./...
GOWORK=off go test -race ./...
GOWORK=off go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
```

`go test ./...` is the broadest regression check. `go test -race ./...` is the stronger concurrency
check and should be run before a release candidate is cut. `govulncheck` is the minimum
vulnerability scan expected before dispatching the release workflow.

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
- Do not mark migration-ledger rows or coverage rows as `verified` without the matching test
  evidence.
- Record the exact `ci`, `stress`, `runtime-tests`, and `runtime-native` run IDs that fed the
  release workflow input set.
