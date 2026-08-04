# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Removed

- The v0.1 compatibility facade, completing its scheduled v0.3 removal: `Codex`/`NewCodex`, the public
  `CodexExec` transport, `NewThread`, `Thread.Run`, `Thread.RunStreamed`, facade-only response
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
