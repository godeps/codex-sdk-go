# Migrating The v0.1 Facade To The Context-First API

Status: the v0.1 facade shipped during the v0.2.x migration window and is removed from the v0.3
API surface. v0.3 uses one long-lived app-server transport and one context-first `Client` API.

## Required Replacements

| Removed API | v0.3 replacement |
|---|---|
| `NewCodex(options)` | `NewClient(ctx, options)` |
| `Codex.StartThread(options)` | `Client.StartThread(ctx, options)` |
| `Codex.ResumeThread(id, options)` | `Client.ResumeThread(ctx, id, options)` |
| `Thread.Run(input, options)` | `Thread.RunContext(ctx, input, options)` |
| `Thread.RunStreamed(input, options)` | `Thread.StartTurnContext(ctx, input, options)` then `TurnHandle.StreamContext` or `RunContext` |
| `TurnHandle.Run()` | `TurnHandle.RunContext(ctx)` |
| `Thread.Read(includeTurns)` | `Thread.ReadContext(ctx, includeTurns)` |
| `Thread.SetName(name)` | `Thread.SetNameContext(ctx, name)` |
| `Thread.Compact()` | `Thread.CompactContext(ctx)` |
| `Codex.Models(includeHidden)` | `Client.ListModels(ctx, includeHidden)` |
| `Codex.LoginAPIKey(apiKey)` | `Client.LoginAPIKey(ctx, apiKey)` |
| `Codex.StartChatGPTLogin()` | `Client.LoginChatGPT(ctx)` |
| `Codex.StartChatGPTDeviceCodeLogin()` | `Client.LoginDeviceCode(ctx)` |
| login `Wait()` / `Cancel()` | `WaitContext(ctx)` / `CancelContext(ctx)` |
| `Codex.Account(refreshToken)` | `Client.Account(ctx, refreshToken)` |
| `Codex.Logout()` | `Client.Logout(ctx)` |
| `CodexExec`, `NewCodexExec`, `NewThread` | `NewClient` plus `WithCodexPath`/managed runtime options |

`CodexOptions`, `ThreadOptions`, `TurnOptions`, input constructors, item types, approval/sandbox
types, generated protocol packages, goals, and runtime packaging remain supported. `CodexOptions`
can still be passed as an `Option`; callers may also use the `With*` helpers.

## Behavioral Changes

- `NewClient` initializes the app-server immediately and returns startup errors directly.
- One app-server process is shared across threads and turns; the process-per-call `codex exec`
  transport no longer exists.
- Blocking facade helpers no longer inject `context.Background()`. Callers choose cancellation and
  deadlines explicitly.
- Turn streaming is pull-based and single-consumer. Read with `TurnStream.Next(ctx)` and close the
  stream when abandoning it.
- `TurnResult` replaces the facade `Turn` and includes ID, status, timestamps, duration, error,
  items, final response, and usage.
- Runtime resolution is explicit path, `CODEX_RUNTIME_PATH`, verified managed cache, then optional
  PATH fallback enabled by `WithAllowPATH(true)`.

## Upgrade Checklist

1. Replace the root constructor and retain the returned initialization error.
2. Thread a caller-owned `context.Context` through thread, turn, login, account, and goal calls.
3. Replace channel-based `RunStreamed` consumers with the pull stream.
4. Replace facade result types with `TurnResult`, `ModelPage`, `ThreadRecord`, `AccountState`, and
   `LoginResult`.
5. Replace direct `CodexExec` use with a configured `Client`.
6. Run `go test ./...` and `go vet ./...`; compile failures naming removed symbols identify all
   remaining migration work.

The v0.2.0 compatibility evidence remains recorded in the parity ledger as release history. The
v0.3 absence test is `TestLegacyFacadeIsAbsentFromPublicAPI`.
