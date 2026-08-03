# Contributing

Keep changes small, evidence-based, and aligned with the checked-in API surface.

## Before Editing Docs Or Examples

- Read `go doc -all .` from the repo root with `GOWORK=off`.
- Prefer the current exported API over old examples or planning text.
- Do not mark any plan row as verified without evidence.
- Do not touch implementation files when the task is documentation-only.

## Verification

Run the commands that prove the docs still match the code:

```bash
GOWORK=off go doc -all .
GOWORK=off go test ./example/...
GOWORK=off go test ./cmd/codex-sdk-gen
GOWORK=off go test ./cmd/codex-sdk-runtime
GOWORK=off go test ./internal/runtimebin ./internal/router
GOWORK=off go test ./...
GOWORK=off go test -race ./...
```

For runtime packaging and lock updates:

```bash
GOWORK=off go run ./cmd/codex-sdk-runtime version 0.144.4
GOWORK=off go run ./cmd/codex-sdk-gen verify
```

`cmd/codex-sdk-gen verify` requires the pinned reference checkout described in
`docs/codex-sdk-parity/README.md`.

## Docs To Keep In Sync

- `README.md`
- `README_CN.md`
- `docs/api-reference.md`
- `docs/runtime-installation.md`
- `docs/auth-approval-goal.md`
- `docs/concurrency-cancellation.md`
- `docs/generated-protocol-policy.md`
- `docs/migration-v0.1-to-v0.2.md`
- `docs/release-verification.md`
- `docs/supported-platforms.md`
- `docs/codex-sdk-parity/README.md`
- `docs/codex-sdk-parity/compatibility-ledger.md`
- `docs/codex-sdk-parity/acceptance-test-spec.md`
- `docs/codex-sdk-parity/coverage-matrix.md`
- `example/common`
- `example/stream`
- `example/steer`
- `example/image`
- `example/structured-output`
- `CHANGELOG.md`

If you change a doc example, make sure the code still compiles against the current exported API.
