# Supported Platforms

The runtime release pipeline targets the following GOOS/GOARCH combinations.

| GOOS | GOARCH | target triple | executable |
|---|---|---|---|
| darwin | amd64 | x86_64-apple-darwin | codex |
| darwin | arm64 | aarch64-apple-darwin | codex |
| linux | amd64 | x86_64-unknown-linux-musl | codex |
| linux | arm64 | aarch64-unknown-linux-musl | codex |
| windows | amd64 | x86_64-pc-windows-msvc | codex.exe |
| windows | arm64 | aarch64-pc-windows-msvc | codex.exe |

## What The Table Means

- The target triple is the runtime packaging identity used by `cmd/codex-sdk-runtime`.
- The executable name is the binary that the managed runtime installer expects inside the archive.
- Native release evidence is required for each entry in the parity and release process.

## Related Commands

- `GOWORK=off go run ./cmd/codex-sdk-runtime version 0.144.4`
- `GOWORK=off go run ./cmd/codex-sdk-runtime install ...`
- `GOWORK=off go run ./cmd/codex-sdk-runtime path ...`
