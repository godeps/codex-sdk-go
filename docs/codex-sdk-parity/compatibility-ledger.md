# v0.1 compatibility contract and migration ledger

Status: final v0.2.0 release ledger. Every preserved behavior and intentional difference below has
checked-in acceptance evidence.

This ledger defines what the parity rebuild preserves for existing Go SDK consumers. An observed
change not listed here is a release blocker until it records the old behavior, new behavior,
migration, first release, and acceptance evidence.

| Area | Old v0.1 behavior | New v0.2 behavior / intentional difference | Migration | First release | Acceptance evidence |
|---|---|---|---|---|---|
| Public API | Exported constructors, options, inputs, items, `Run`, and `RunStreamed` compile. | The source-compatible facade remains, now explicitly deprecated; the context-first `Client` is preferred. | Follow the symbol map below; no immediate source edit is required. | v0.2.0 | `TestCompatibilityPublicAPISnapshotMatchesV01Golden`, `TestCompatibilityCompileFixturesBuildAgainstCurrentPublicAPI` |
| CLI lookup | Resolution is explicit option, then PATH, then `vendor/<target-triple>/codex`. | The facade preserves that order. `Client` additionally supports a signed cache/runtime policy and requires explicit `WithAllowPATH` when PATH fallback is desired. | Pin with `WithCodexPath` or `WithRuntimeCacheRoot`; opt into PATH with `WithAllowPATH`. | v0.2.0 | `TestCompatibilityFindCodexPathPrefersPATHBinary`, `TestCompatibilityFindCodexPathFallsBackToBundledVendorPath`, runtime resolver tests |
| Environment | Nil `Env` inherits; non-nil `Env` fully replaces the inherited environment. | Unchanged, including case-insensitive Windows PATH handling. | None. | v0.2.0 | `TestCompatibilityNilEnvInheritsParentAndSetsOriginator`, `TestCompatibilityOverrideEnvReplacesParentEnvironment` |
| Endpoint/auth | Base URL, API key, and originator overrides reach the child. | Observable behavior is unchanged; secrets are additionally redacted from captured stderr. | None. | v0.2.0 | `TestCompatibilityCLIArgvTranscriptMatchesV01Golden`, app-server redaction tests |
| Configuration | Option/config/TOML precedence and serialization are stable. | Observable facade precedence is unchanged; app-server options are mapped directly. | None. | v0.2.0 | `TestCompatibilityCLIArgvTranscriptMatchesV01Golden` and compatibility transcript goldens |
| Output schema | Only object schemas are accepted; the temporary schema file lives through execution and is removed afterward. | Facade behavior is unchanged; the app-server API sends the schema JSON directly. | New callers pass the same `TurnOptions.OutputSchema` through `RunContext`. | v0.2.0 | `TestCompatibilityOutputSchemaRejectsNonObjects`, `TestCompatibilityOutputSchemaWritesStableJSONAndCleansUp`, `TestCompatibilityRunStreamedPreservesSchemaLifetimeDuringRun` |
| Thread lifecycle | Start/resume exposes a mutable thread ID. | Source and output are unchanged; ID access is synchronized to remove the v0.1 data race. | None. | v0.2.0 | compile fixtures, `go test -race`, thread lifecycle tests |
| Run | A process per call produces events, final response, usage, cancellation, and process errors. | The facade preserves results while adapting to one persistent app-server transport. | Prefer `Thread.RunContext` for explicit cancellation and lifecycle ownership. | v0.2.0 | `TestCompatibilityRunCollectsItemsUsageAndFinalResponse`, `TestCompatibilityRunReturnsTurnFailedMessage` |
| RunStreamed | Events remain ordered and terminate with one result/error. | The adapter is bounded and closes its iterator when abandoned, preventing goroutine leaks. | Prefer `StartTurnContext` plus `TurnHandle.Stream` or `RunContext`. | v0.2.0 | `TestCompatibilityRunStreamedPreservesSchemaLifetimeDuringRun`, `TestRunStreamedAbandonedConsumerClosesBoundedAdapter` |
| Inputs | Text and local-image constructors serialize as before. | Those inputs are unchanged; data URL, skill, and mention inputs are added. | Existing inputs need no change; use the new constructors only when needed. | v0.2.0 | `TestCompatibilityItemsInputMapsTextToStdinAndImagesToArgs`, input golden tests |
| Unknown data | `UnknownItem.Raw` preserves original JSON bytes and meaning. | Preserved and extended to generated discriminated unions, notifications, primitives, and nested unions. | Continue inspecting `Raw`; new code may also use generated raw fallback variants. | v0.2.0 | `TestThreadEventUnmarshalUnknownItem`, `TestUnknownDiscriminatorThreadItemFallsBackToRawRoundTrip`, `TestUnknownDiscriminatorNotificationFallsBackToRawRoundTrip`, `TestNestedUnionUnknownVariantFallsBackToRaw` |

## Release artifacts

- `testdata/compat/v0.1/public-api.txt`: normalized public API snapshot.
- `testdata/compat/v0.1/compile/`: current README examples and representative consumers.
- `testdata/compat/v0.1/transcripts/`: argv, environment, JSONL, structured-output, and failure
  goldens.
- the final release evidence bundle links exact-SHA CI, stress, native-runtime, and runtime-test runs.

## Deprecation support window

The v0.1 facade is supported and regression-tested for every v0.2.x release. It will not be removed
before v0.3.0. Removal requires a separate approved compatibility plan, a release-note migration
notice, and a major-version-compatible decision if the module has reached v1 at that time.

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
