# Migration From v0.1 To The Current Context-First API

The module still ships the v0.1 facade, but new code should use the context-first `Client`
surface.

The v0.1 facade is supported and regression-tested throughout the v0.2.x release line. It will not
be removed before v0.3.0; removal requires a separately approved compatibility plan and release
notice. This gives existing consumers a full minor release line to migrate.

## Core Mapping

| v0.1 symbol | Current API |
|---|---|
| `NewCodex(options)` | `NewClient(ctx, options)` |
| `Codex.StartThread(options)` | `Client.StartThread(ctx, options)` |
| `Codex.ResumeThread(id, options)` | `Client.ResumeThread(ctx, id, options)` |
| `Thread.Run(input, turnOptions)` | `Thread.RunContext(ctx, input, turnOptions)` |
| `Thread.RunStreamed(input, turnOptions)` | `Thread.StartTurnContext(ctx, input, turnOptions)` + `TurnHandle.Stream` / `RunContext` |
| `Thread.Read(includeTurns)` | `Thread.ReadContext(ctx, includeTurns)` |
| `Thread.SetName(name)` | `Thread.SetNameContext(ctx, name)` |
| `Thread.Compact()` | `Thread.CompactContext(ctx)` |
| `Codex.Models(includeHidden)` | `Client.ListModels(ctx, includeHidden)` |
| `Codex.LoginAPIKey(apiKey)` | `Client.LoginAPIKey(ctx, apiKey)` |
| `Codex.StartChatGPTLogin()` | `Client.LoginChatGPT(ctx)` |
| `Codex.StartChatGPTDeviceCodeLogin()` | `Client.LoginDeviceCode(ctx)` |
| `Codex.Account(refreshToken)` | `Client.Account(ctx, refreshToken)` |
| `Codex.Logout()` | `Client.Logout(ctx)` |

## Behavioral Differences

- `Client` starts the app-server immediately and owns its lifecycle explicitly.
- `TurnHandle.Stream` is pull-based and single-consumer.
- `TurnHandle.SteerContext` and `InterruptContext` let you act on one live turn without using the
  compatibility wrapper.
- `TurnResult` exposes timestamps, duration, error, items, final response, and usage in one result.

## What Stayed Compatible

The compatibility facade keeps the older call shapes working:

- `NewCodex`
- `Codex.StartThread`
- `Codex.ResumeThread`
- `Thread.Run`
- `Thread.RunStreamed`
- `Thread.Read`
- `Thread.SetName`
- `Thread.Compact`

Use the facade only when you need to keep older call sites compiling during migration.

## Suggested Migration Order

1. Switch callers to `NewClient(ctx, ...)`.
2. Replace `Run` and `RunStreamed` with `RunContext` or `StartTurnContext` plus `TurnHandle`.
3. Replace thread helpers with the context-first `Thread` methods.
4. Replace login `Wait` / `Cancel` calls with `WaitContext` / `CancelContext`.
5. Once all call sites are converted, remove reliance on the compatibility facade.
