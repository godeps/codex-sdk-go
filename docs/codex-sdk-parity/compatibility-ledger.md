# v0.1 compatibility contract and migration ledger

Status: baseline contract frozen for implementation; evidence fixtures are created in Phase 0.

This ledger defines what the parity rebuild must preserve for existing Go SDK consumers. Each row
must gain fixture/test evidence before the old exec transport is replaced. Any observed change not
listed here is a release blocker until documented and approved.

| Area | v0.1 contract to preserve | Planned proof | Intentional difference |
|---|---|---|---|
| Public API | Existing exported constructors, options, input helpers, item/result types, `Run`, and `RunStreamed` compile | normalized `go doc -all` snapshot plus compile fixtures | Deprecation comments only; no removal |
| CLI lookup | explicit CLI option, then PATH, then `vendor/<target-triple>/codex` | process-launch transcript per source | New API requires explicit PATH policy; deprecated facade retains default PATH lookup |
| Environment | nil `Env` inherits; non-nil `Env` fully replaces inherited environment | child-process environment transcript, including Windows `PATH`/`Path` | None |
| Endpoint/auth | BaseURL, API key, and originator overrides reach the child exactly as today | arguments/environment transcript with redacted canaries | None |
| Configuration | option/config/TOML precedence and serialization remain stable | golden argv/config transcript | None |
| Output schema | object validation, temporary-file contents/lifetime, and cleanup remain stable | valid/invalid schema and cleanup tests | New app-server API may send JSON directly; facade observable behavior stays unchanged |
| Thread lifecycle | start/resume behavior and exposed thread ID remain stable | fake-runtime transcript and compile fixture | Thread ID storage becomes race-free; this is a defect fix |
| Run | event collection, final response selection, usage, cancellation, and process errors remain stable | JSONL transcript goldens | Internally adapted to persistent app-server |
| RunStreamed | event ordering, terminal result/error, and cancellation remain stable | bounded-stream transcript and abandoned-consumer test | Adapter is bounded and closes its iterator to prevent leaks |
| Inputs | text and local-image constructors and serialization remain stable | input golden vectors | New API additionally supports data URL, skill, and mention |
| Unknown data | `UnknownItem.Raw` retains the original bytes/meaning | unknown-item round-trip golden | New protocol types add raw fallbacks beyond old items |

## Required baseline artifacts

- `testdata/compat/v0.1/public-api.txt`: normalized public API snapshot.
- `testdata/compat/v0.1/compile/`: current README examples and representative consumers.
- `testdata/compat/v0.1/transcripts/`: argv, environment, JSONL, structured-output, and failure
  goldens.
- one test/evidence link per ledger row before its status may become `verified`.

## Migration map

The legacy facade remains in place so downstream consumers can migrate gradually.

| v0.1 symbol | Current API | Notes |
|---|---|---|
| `NewCodex` | `NewClient(ctx, options)` | New code should own context and lifecycle explicitly. |
| `Codex.StartThread` | `Client.StartThread` | Both create a persisted thread. |
| `Codex.ResumeThread` | `Client.ResumeThread` | Both resume by thread ID. |
| `Thread.Run` | `Thread.RunContext` | The context-first API returns `TurnResult`. |
| `Thread.RunStreamed` | `Thread.StartTurnContext` + `TurnHandle.Stream`/`RunContext` | The handle stream is single-consumer. |
| `Thread.Read` | `Thread.ReadContext` | The context-first API returns `ThreadRecord`. |
| `Thread.SetName` | `Thread.SetNameContext` | Same server method, explicit context. |
| `Thread.Compact` | `Thread.CompactContext` | Same server method, explicit context. |
| `Codex.Models` | `Client.ListModels` | The root client is concurrency-safe. |
| `Codex.LoginAPIKey` | `Client.LoginAPIKey` | No behavior change. |
| `Codex.StartChatGPTLogin` | `Client.LoginChatGPT` | The handle keeps `WaitContext` and `CancelContext`. |
| `Codex.StartChatGPTDeviceCodeLogin` | `Client.LoginDeviceCode` | Same handle contract as browser login. |
| `Codex.Account` | `Client.Account` | Context-first and returns `AccountState`. |
| `Codex.Logout` | `Client.Logout` | Same server method, explicit context. |

## Change procedure

For every necessary incompatibility, add the old behavior, new behavior, migration instructions,
first release version, rationale, and acceptance evidence to this ledger before merging it. Removal
of deprecated symbols requires a separate approved plan and release decision.
