# Codex SDK (Go)

The Go SDK exposes the Codex app-server in two layers:

- `Client`, the preferred context-first API for long-lived, concurrent use.
- `Codex`, the legacy v0.1 compatibility facade for existing callers.

Use `Client` for new code. Keep `Codex` only while migrating older call sites.

## Getting Started

```go
package main

import (
	"context"
	"fmt"
	"time"

	codex "github.com/godeps/codex-sdk-go"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := codex.NewClient(ctx)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	thread, err := client.StartThread(ctx, codex.ThreadOptions{
		WorkingDirectory: ".",
		SandboxMode:      codex.SandboxDangerFullAccess,
		ApprovalPolicy:   codex.ApprovalNever,
	})
	if err != nil {
		panic(err)
	}

	turn, err := thread.RunContext(ctx, codex.TextInput("Summarize this repository"), codex.TurnOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Println(turn.FinalResponse)
}
```

`NewClient` starts the app-server immediately. Add options such as `codex.WithCodexPath`,
`codex.WithRuntimeCacheRoot`, `codex.WithRuntimeVersion`, `codex.WithAllowPATH`,
`codex.WithBaseURL`, or `codex.WithAPIKey` when you need to pin runtime resolution or override
auth/config settings.

## API Overview

### Root client

- `NewClient(ctx, opts...)` starts and initializes the app-server immediately.
- `Client.Metadata`, `Close`, `CloseContext`, `Wait`, and `WaitContext` expose lifecycle state.
- `Client.ListModels`, `StartThread`, `ResumeThread`, `ReadThread`, `ListThreads`, `ForkThread`,
  `ArchiveThread`, `UnarchiveThread`, `SetThreadName`, and `CompactThread` use
  `context.Context`.
- `Client.LoginAPIKey`, `LoginChatGPT`, `LoginDeviceCode`, `Account`, `Logout`, `GetGoal`,
  `SetGoal`, `ClearGoal`, `PauseGoal`, and `StartGoal` use the same context-first style.

### Threads, turns, and goals

- `Thread.RunContext` is the direct "start and collect" path for one turn.
- `Thread.StartTurnContext` returns a live `TurnHandle`.
- `TurnHandle.Stream`, `RunContext`, `Steer`, and `Interrupt` control one in-flight turn.
- `Thread.StartGoalContext` returns a `GoalHandle`.
- `GoalHandle.StreamContext`, `RunContext`, `CancelContext`, and `Close` control one logical goal.

`TurnResult` carries the collected result for the context-first API: turn ID, status, error,
timestamps, duration, final response, items, and usage.

### Inputs

Use `TextInput` for plain prompts or `ItemsInput` for structured input. The helper constructors
map to the current app-server wire format:

- `DataURLImageInput(url string)`
- `LocalImageInput(path string)`
- `SkillInput(name, path string)`
- `MentionInput(name, path string)`

`ItemsInput` preserves order. Invalid, empty, or unsupported entries return an error instead of
being dropped.

### Options

`NewClient` accepts either a `CodexOptions` value directly or helper `Option` values. The current
options include:

- `WithCodexPath`
- `WithBaseURL`
- `WithAPIKey`
- `WithConfig`
- `WithEnv`
- `WithRuntimeCacheRoot`
- `WithRuntimeVersion`
- `WithAllowPATH`

`ThreadOptions` and `TurnOptions` keep the current app-server fields visible in Go.

Common enum values:

- `ApprovalMode`: `never`, `on-request`, `on-failure`, `untrusted`
- `ApprovalPreset`: `deny_all`, `auto_review`
- `SandboxMode`: `read-only`, `workspace-write`, `danger-full-access`
- `ModelReasoningEffort`: `minimal`, `low`, `medium`, `high`, `xhigh`
- `WebSearchMode`: `disabled`, `cached`, `live`
- `ReasoningSummary`: `none`, `auto`, `brief`, `detailed`

`ApprovalPresetDenyAll` maps to `ApprovalNever`. `ApprovalPresetAutoReview` maps to
`ApprovalOnRequest` with the `auto_review` reviewer.

## Streaming, Steering, And Interrupting

`TurnHandle.Stream()` returns a pull-based `TurnStream`. Each `Next(ctx)` call yields one routed
`ThreadEvent`. `Close()` is idempotent and unregisters the route. After `Close`, future `Next`
calls return `ErrStreamClosed`.

Use `TurnHandle.Steer` to inject more input into the active turn and `TurnHandle.Interrupt` to
stop it.

## Login, Account, And Goal

The context-first login and account APIs are:

- `Client.LoginAPIKey(ctx, apiKey)`
- `Client.LoginChatGPT(ctx)`
- `Client.LoginDeviceCode(ctx)`
- `Client.Account(ctx, refreshToken)`
- `Client.Logout(ctx)`

The interactive login handles expose `WaitContext` and `CancelContext`. The legacy facade keeps
the older `Wait` and `Cancel` helpers for compatibility.

Goal APIs operate on persisted threads:

- `Thread.GetGoalContext(ctx)`
- `Thread.SetGoalContext(ctx, update)`
- `Thread.ClearGoalContext(ctx)`
- `Thread.PauseGoalContext(ctx)`
- `Thread.StartGoalContext(ctx, objective)`

`GoalHandle` is the logical goal stream. Use `RunContext` to collect the final goal result or
`CancelContext` for best-effort pause and interrupt cleanup.

## Runtime Installation And Resolution

The SDK and the runtime installer solve different problems:

- `NewClient` resolves a Codex executable from `WithCodexPath` first, then from the
  `CODEX_RUNTIME_PATH` environment variable, then from a verified managed runtime cache, and then
  from `PATH` only when `WithAllowPATH(true)` is set.
- `WithEnv` replaces the environment passed to the Codex CLI process. If you do not set it, the
  SDK inherits the current process environment.
- `codex-sdk-runtime` manages pinned runtime archives in a cache. Its resolution order is explicit
  binary path, `CODEX_RUNTIME_PATH`, verified managed cache, and then `PATH` only when allowed by
  policy.

Supported runtime targets:

| GOOS | GOARCH | target triple | executable |
|---|---|---|---|
| darwin | amd64 | x86_64-apple-darwin | codex |
| darwin | arm64 | aarch64-apple-darwin | codex |
| linux | amd64 | x86_64-unknown-linux-musl | codex |
| linux | arm64 | aarch64-unknown-linux-musl | codex |
| windows | amd64 | x86_64-pc-windows-msvc | codex.exe |
| windows | arm64 | aarch64-pc-windows-msvc | codex.exe |

The checked-in runtime artifacts under `runtime/testdata/release/` make the installer runnable
offline on the matching platform. For the current platform, this command installs from the
versioned local archive and manifest:

```bash
archive="runtime/testdata/release/codex-sdk-go-runtime_0.144.4_$(go env GOOS)_$(go env GOARCH).tar.gz"
cache="$(mktemp -d)"
GOWORK=off go run ./cmd/codex-sdk-runtime install \
  --cache-root "$cache" \
  --manifest runtime/manifest.json \
  --manifest-signature runtime/manifest.json.sig \
  --archive "$archive"
GOWORK=off go run ./cmd/codex-sdk-runtime path \
  --cache-root "$cache" \
  --runtime-version 0.144.4
```

`version` converts an SDK/runtime version to the upstream release tag:

```bash
GOWORK=off go run ./cmd/codex-sdk-runtime version 0.144.4
```

## Protocol Generation And Locking

`reference.lock.json` pins the upstream Codex checkout, Python root, runtime package/version,
aggregate schema path, schema SHA-256, and the six supported target triples. The generator refuses
to silently switch to another checkout.

- `GOWORK=off go run ./cmd/codex-sdk-gen verify` checks the checked-in schema and
  `protocol/manifest.json` against the lock.
- `GOWORK=off go run ./cmd/codex-sdk-gen refresh` refreshes the snapshot from the pinned reference
  checkout when the lock changes.

If the pinned reference repository is not available, the generator fails with a diagnostic instead
of guessing.

## Migration And Examples

The legacy facade remains in the module so callers can migrate gradually. See
[docs/migration-v0.1-to-v0.2.md](docs/migration-v0.1-to-v0.2.md) for the compatibility map and
[docs/api-reference.md](docs/api-reference.md) for the current context-first surface.

Executable examples live under:

- `example/common`
- `example/stream`
- `example/steer`
- `example/image`
- `example/structured-output`

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT. See [LICENSE](LICENSE).
