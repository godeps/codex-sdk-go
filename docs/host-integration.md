# Embedding the managed client in a host

The context-first `NewClient` API supports host-owned approvals and process supervision.

## Server requests

`WithServerRequestHandler(handler)` installs a public `ServerRequestHandler`:

```go
type ServerRequestHandler func(
    ctx context.Context,
    method string,
    params json.RawMessage,
) (json.RawMessage, error)
```

The transport keeps the original JSON-RPC request ID and sends the handler's result back to that exact request. The handler receives an independent copy of the parameters. It can be called concurrently; blocking one request does not block the protocol reader. The context is cancelled when the client closes.

Return the result schema required by the specific method. A command execution approval is different from a permissions request, tool call, user input request or token refresh. Unknown or unsupported methods should return an error. An error becomes an RPC error response; it does not grant permission. A nil handler keeps the built-in deny-by-default behavior.

Applications forwarding requests to another UI must preserve their run, native thread/turn/item and request identity, and must reject late or mismatched answers. Installing this handler does not itself configure the thread's approval policy.

## Process supervision

`WithCommandHook(func(*exec.Cmd) error)` runs after environment, working directory and pipes are configured, immediately before the managed app-server starts. A host can configure platform-specific process groups or job objects and capture the command for its supervisor. Returning an error prevents launch.

The hook must not call `Start`, `Run` or `Wait`, and must not replace the SDK's protocol pipes. The SDK owns start and wait. A host managing descendants must implement and verify the corresponding group/job termination; merely cancelling an HTTP request is not evidence that those processes stopped.

## Environment

`WithEnv(nil)` retains parent inheritance. A non-nil environment, including an empty map, replaces the parent environment. SDK-managed originator and explicitly configured API/base URL variables are still applied. Managed runtime path directories may also be added as required.

The app-server transport honors this replacement all the way to the actual child process. Hosts should explicitly select the environment and credentials appropriate to their local agent. The command hook sees the final environment and can intentionally modify it.

## Validation

`host_integration_test.go` launches a real OS child speaking the stdio protocol to verify public handler correlation, denial and error behavior, shutdown cancellation, pre-start rejection, and parent-environment isolation. `thread_config_test.go` verifies that individual thread options work without an explicitly supplied Config map. These are protocol fixtures, not provider-backed model tests.
