# Concurrency And Cancellation

`Client` is designed for concurrent use. The turn, login, and goal handle objects are not.

## Concurrent Objects

Safe to share across goroutines:

- `Client`
- `Client.Metadata()`
- `Client.ListModels`, `StartThread`, `ResumeThread`, and the other root client methods

Single-consumer or single-owner objects:

- `TurnHandle`
- `TurnStream`
- `GoalHandle`
- `GoalStream`
- login handles returned by `LoginChatGPT` and `LoginDeviceCode`

## Cancellation

All blocking APIs accept `context.Context` where cancellation matters:

- `Client.CloseContext`
- `Client.WaitContext`
- `Thread.RunContext`
- `Thread.StartTurnContext`
- `TurnHandle.StreamContext`
- `TurnHandle.RunContext`
- `TurnHandle.SteerContext`
- `TurnHandle.InterruptContext`
- goal and login `*Context` methods

If the context is canceled, the method returns the context error. The SDK does not silently retry
canceled operations.

## Stream Lifecycle

`TurnStream.Next(ctx)` returns:

- one routed event per call
- `io.EOF` after normal completion
- `ErrStreamClosed` after the stream is closed
- a transport or context error when the underlying operation fails

Call `Close()` when you are done. `Close()` is idempotent and unregisters the route.

## Turn Steering And Interrupts

`TurnHandle.Steer` injects extra input into an active turn. `TurnHandle.Interrupt` requests
interruption of the active turn. These calls are best used while the turn is still running.

## Goal Cancellation

`GoalHandle.CancelContext` performs best-effort pause and interrupt cleanup. It is the preferred
way to abandon a logical goal because it closes the route and clears SDK-side state.

## Practical Rule

If one goroutine owns a turn, login, or goal handle, keep all direct method calls on that handle in
one place. Share only the parent `Client` across goroutines.
