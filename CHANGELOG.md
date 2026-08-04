# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

## [0.3.1] - 2026-08-03

### Added

- Root package docs, executable package examples, `llms.txt`, `govulncheck` in CI, CodeQL, maintained
  Node 24 GitHub Actions, and an evidence-gated release flow that creates the tag from `release.yml`.
- Deterministic coverage for configuration flattening, environment precedence, output-schema
  validation, retry timing, and materialized thread behavior.

### Changed

- `Thread` now retains only the immutable app-server thread identity and client reference; dead
  lazy-materialization state and locking left by the v0.1 facade are gone.
- Release verification now documents the required workflow run IDs and the post-evidence tag
  creation flow for `v0.3.1` and later.

## [0.3.0] - 2026-08-03

### Removed

- The legacy v0.1 surface, completing its scheduled v0.3 removal: `Codex`/`NewCodex`, the public
  `CodexExec` transport, `NewThread`, `Thread.Run`, `Thread.RunStreamed`, legacy-only response
  types, and login/thread helpers without explicit context.
- The process-per-turn `codex exec` implementation and its v0.1 compile/transcript fixtures. The
  SDK now has one app-server transport and one context-first public API.

### Added

- Current API reference, runtime installation guide, auth/approval/goal guide,
  concurrency/cancellation guide, generated protocol policy, migration guide,
  supported-platform guide, and release verification guide.
- Executable example entrypoints under `example/common`, `example/stream`, `example/steer`,
  `example/image`, and `example/structured-output`.

### Changed

- README and contributing guidance now describe the current option-based `NewClient` API and the
  current runtime resolution order.

[Unreleased]: https://github.com/godeps/codex-sdk-go/compare/v0.3.1...HEAD
[0.3.1]: https://github.com/godeps/codex-sdk-go/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/godeps/codex-sdk-go/compare/v0.2.0...v0.3.0
