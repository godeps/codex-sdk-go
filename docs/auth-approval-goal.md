# Auth, Approval, And Goal Management

This SDK keeps authentication, tool approval, and goal state as separate surfaces.

## Authentication

`Client` supports four auth-related flows:

- `LoginAPIKey(ctx, apiKey)` for direct API-key login
- `LoginChatGPT(ctx)` for browser-based login
- `LoginDeviceCode(ctx)` for device-code login
- `Account(ctx, refreshToken)` and `Logout(ctx)` for runtime account state

Browser and device-code login return handle objects. Use `WaitContext` to block until completion
and `CancelContext` to abort the attempt.

## Approval

The thread and turn options expose both the legacy approval mode and the higher-level approval
preset:

- `ApprovalMode`: `never`, `on-request`, `on-failure`, `untrusted`
- `ApprovalPreset`: `deny_all`, `auto_review`

Current mapping:

| Preset | Result |
|---|---|
| `ApprovalPresetDenyAll` | `ApprovalNever` |
| `ApprovalPresetAutoReview` | `ApprovalOnRequest` with `auto_review` reviewer |

When both are present, the preset wins. Use the legacy `ApprovalPolicy` only when you are
migrating code that already speaks `ApprovalMode`.

## Goal State

Goals are attached to persisted threads. The root client exposes:

- `GetGoal(ctx, threadID)`
- `SetGoal(ctx, threadID, update)`
- `ClearGoal(ctx, threadID)`
- `PauseGoal(ctx, threadID)`
- `StartGoal(ctx, threadID, objective)`

Thread-scoped helpers expose the same operations on the current thread:

- `Thread.GetGoalContext(ctx)`
- `Thread.SetGoalContext(ctx, update)`
- `Thread.ClearGoalContext(ctx)`
- `Thread.PauseGoalContext(ctx)`
- `Thread.StartGoalContext(ctx, objective)`

`GoalHandle` represents the logical goal stream. Use `RunContext` to collect the final result or
`CancelContext` for best-effort pause and interrupt cleanup.

## When To Use Which Surface

- Use `LoginAPIKey` when the caller already owns an API key.
- Use `LoginChatGPT` or `LoginDeviceCode` when the runtime should drive interactive user login.
- Use `ApprovalPreset` when you want the higher-level policy mapping used by the current runtime.
- Use the goal helpers when you need a persisted objective with streamable progress across turns.
