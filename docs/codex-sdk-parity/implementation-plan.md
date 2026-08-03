# Full Codex SDK parity implementation plan

Status: implementation and candidate verification complete; final exact-SHA release workflow is the publishing gate
Reference Python revision: `bb5054fe47`
Pinned Codex runtime: `0.144.4`

## 1. Objective

Rebuild this module as a production Go client for `codex app-server --listen stdio://` with:

1. all protocol definitions emitted by the pinned runtime's aggregate v2 JSON Schema;
2. complete public Python SDK behavior, including thread and turn lifecycle, streaming,
   steering, interruption, account/login flows, approval requests, goals, model discovery,
   typed errors, retry, and forward-compatible unknown payloads;
3. runtime acquisition and packaging for Darwin/Linux/Windows on amd64 and arm64;
4. compatibility for the existing v0.1 Go API during migration;
5. deterministic generation, comprehensive tests, release provenance, and documented operations.

“Parity” means equivalent observable behavior and wire compatibility. It does not require Python
class-for-class API duplication: Go uses `context.Context`, goroutines, typed channels, and one
concurrency-safe client instead of separate sync/async clients.

## 2. Baseline and constraints

The existing Go client constructs lightweight thread values in `codex.go:10-25`, starts one
`codex exec --experimental-json` process per turn in `exec.go:81-147`, and parses a small manual
JSONL event union in `types.go:19-270`. The new implementation cannot achieve app-server parity by
extending that transport.

The Python reference:

- owns a long-lived app-server process and initialize handshake in `client.py:212-309`;
- multiplexes request responses and login/turn/goal notifications in
  `_message_router.py:17-278`;
- exposes thread lifecycle and turn handles in `api.py:75-807`;
- generates protocol models and notification metadata from runtime schemas in
  `scripts/update_sdk_artifacts.py:530-858`;
- pins runtime version `0.144.4` in `pyproject.toml:19`;
- maps six runtime assets in `_runtime_setup.py:99-122`.

Project constraints:

- preserve user work and re-check the baseline before every implementation phase;
- introduce no runtime dependency outside the Go standard library;
- generated code is checked in and reproducible;
- no network access occurs merely by constructing a client;
- all external calls accept cancellation or a documented bounded timeout;
- all queues, buffers, stderr capture, and in-flight request counts have explicit limits;
- unsupported and future protocol variants retain raw JSON instead of crashing the reader loop;
- implementation does not begin until the PRD and test-spec artifacts exist.

### Reproducible reference input

`reference.lock.json` is the single machine-readable reference lock. It records:

- upstream repository `https://github.com/openai/codex` and commit `bb5054fe47`;
- Python root `sdk/python` and runtime package/version `openai-codex-cli-bin==0.144.4`;
- aggregate v2 schema filename and SHA-256, populated during Phase 0;
- the six supported target triples and the corresponding upstream asset names.

Tools resolve the checkout from `CODEX_REFERENCE_REPO`, then `../../other/codex`, then
`_reference/codex`. A candidate is accepted only when its Git `HEAD` equals the locked commit and
its Python/runtime metadata matches the lock. CI always checks out the locked upstream commit into
`_reference/codex`; missing or mismatched inputs fail with a diagnostic and never fall back to a
different revision.

## 3. Architecture decision record

### Decision

Replace the root client's internals with a long-lived stdio JSON-RPC app-server transport. Generate
the complete protocol package from the pinned runtime schema. Publish platform runtime archives
and a checksum manifest separately from the source Go module; install them explicitly into a
managed cache or supply a caller-selected binary path.

### Drivers

1. app-server features such as steer, interrupt, login, approvals, goals, and interleaved turns
   require bidirectional persistent transport;
2. hundreds of evolving protocol definitions cannot be maintained safely by hand;
3. Go modules cannot select and fetch one large platform-native payload as naturally as Python
   platform wheels, so source and runtime artifacts need separate release paths.

### Alternatives considered

#### Extend `codex exec`

Rejected. It cannot receive server requests or control an already-running turn, creates one
process per turn, and has no app-server request/notification correlation.

#### Embed all six binaries in the main Go module

Rejected. Every consumer would download every platform binary; module size, proxy limits,
security review, and release churn would become unreasonable.

#### Download a binary implicitly in `NewClient`

Rejected. Constructors must not cause hidden network writes, and offline/restricted builds need a
deterministic path. Runtime installation is an explicit command/API operation.

### Consequences

- `Client` becomes an owned resource and must expose idempotent `Close`/`Wait` behavior.
- generated code becomes a large but reviewable artifact whose source of truth is schema + pin.
- releases consist of a small source module plus six runtime archives and one signed checksum
  manifest.
- the v0.1 API remains a facade temporarily but inherits app-server lifecycle semantics.

### Follow-ups

- retain the v0.1 compatibility facade for all v0.2.x releases; earliest removal is v0.3.0 and
  requires a separate approved compatibility plan;
- revisit additional architectures only when upstream publishes matching runtime assets;
- update the pin and regenerate in an isolated change, never mixed with handwritten behavior.

## 4. Target package layout

```text
.
├── client.go, options.go, thread.go, turn.go, login.go, goal.go
├── errors.go, inputs.go, compatibility.go
├── protocol/
│   ├── generated_*.go          # checked-in complete v2 schema output
│   ├── unions.go               # generated/custom JSON union dispatch
│   ├── registry.go             # methods, notifications, server requests
│   └── manifest.json           # schema/type/method coverage evidence
├── internal/
│   ├── appserver/              # process, framing, request correlation, shutdown
│   ├── router/                 # notification/login/turn/goal routing
│   ├── runtimebin/             # target mapping, install, verify, resolve, cache
│   └── testserver/             # deterministic fake app-server executable
├── cmd/
│   ├── codex-sdk-gen/          # schema-to-Go generator
│   └── codex-sdk-runtime/      # explicit runtime installer/verifier
├── schema/                     # normalized pinned aggregate schema + metadata
├── runtime/
│   └── manifest.json           # version, targets, URLs, sha256, sizes, layout
├── reference.lock.json         # immutable Python/runtime/schema source coordinates
├── testdata/                   # JSON-RPC transcripts and golden payloads
└── docs/
```

Root-package interfaces should be consumer-focused. Internal transport/process abstractions remain
under `internal/`; generated wire types live in `protocol` so users can inspect and match events
without coupling to implementation details.

## 5. Workstreams and implementation sequence

Dependent phases run sequentially. Tasks inside a phase may run in parallel only when they do not
edit shared generated or public API files.

### Phase 0 — freeze the contract and baseline

Deliverables:

- record Python commit, runtime pin, Go commit, Go version, supported triples, and schema hash;
- create and validate `reference.lock.json`, including deterministic checkout resolution and CI
  checkout of the locked upstream commit;
- inventory all Python root exports, low-level RPC methods, input variants, error classes,
  notification methods, server requests, and behavioral tests;
- produce `protocol/manifest.json` schema for later generator output;
- capture current Go public symbols with `go doc`/compile fixtures;
- preserve v0.1 behavior with black-box regression tests before changing transport.

Exit gate:

- every row in `coverage-matrix.md` has an acceptance ID and planned owner package;
- existing Go tests plus new compatibility tests pass with `GOWORK=off` and `-race`.

### Phase 1 — deterministic complete protocol generation

Implement `cmd/codex-sdk-gen` using standard-library JSON parsing and templates.

Generation pipeline:

1. resolve the exact pinned runtime;
2. run `codex app-server generate-json-schema --out <temp>`;
3. select the aggregate v2 bundle and server request/notification schemas;
4. normalize ordering, titles, nullable-required fields, aliases, and known upstream schema quirks;
5. emit enums as named string types with constants while accepting future string values;
6. emit structs with exact JSON aliases and pointer/optional semantics;
7. emit discriminated union decoders for all union variants plus raw unknown fallback;
8. emit request, response, notification, server-request, and turn-ID registries;
9. emit a manifest containing source schema SHA-256, all definition names, all method names,
   generated symbols, unsupported constructs, and generator version;
10. format generated Go and compare against checked-in output.

No schema definition or method may be silently skipped. Unsupported constructs fail generation and
appear in the manifest. Generated code must not import Python or require Pydantic-like runtime
reflection.

Exit gate:

- manifest reports zero missing definitions, zero duplicate symbols, zero unhandled discriminators,
  and zero unsupported schema constructs;
- a second generation is byte-for-byte identical;
- schema drift CI fails if regeneration changes tracked output.

### Phase 2 — app-server transport and lifecycle

Implement `internal/appserver`:

- launch `codex app-server --listen stdio://` with controlled cwd/env/config;
- drain stderr into a bounded ring;
- serialize newline-delimited JSON-RPC writes under one mutex;
- run exactly one stdout reader goroutine;
- correlate string request IDs to bounded one-shot waiters;
- classify server requests, notifications, success responses, and error responses;
- support concurrent callers without response theft;
- remove canceled waiters and ignore late responses safely;
- fail all waiters/routes exactly once when transport closes;
- implement idempotent `Close`, graceful terminate deadline, forced kill fallback, and `Wait`;
- expose injectable process/clock/ID seams for tests without exporting implementation interfaces.

Default limits must be documented and configurable: maximum line size, stderr lines, in-flight
requests, per-route pending notifications, total pending notifications, and shutdown timeout.
Overflow must fail explicitly; events must never be silently dropped.

The frozen defaults are: 16 MiB maximum JSON line; stderr ring of 400 lines and 2 MiB; 1,024
in-flight RPCs; 256 events and 8 MiB per route; 1,024 events and 32 MiB for global notifications;
4,096 events and 64 MiB total; 64 concurrent server-request handlers; 2 seconds for graceful
shutdown followed by a 2-second forced-kill wait. The first route count/byte overflow closes only
that route with `ErrLimitExceeded`. Line, global, total, or in-flight overflow terminates the
transport and fails waiters with errors matching `ErrLimitExceeded` and `ErrTransportClosed`.

Exit gate:

- transport unit, fuzz, cancellation, malformed-input, overflow, early-exit, and concurrency tests
  pass under `go test -race`;
- 1,000 mixed concurrent fake-server requests complete with correct response correlation;
- all goroutines and child processes terminate after every close/error path.

### Phase 3 — router, errors, retry, and approval requests

Implement routing equivalent to Python behavior:

- global notification stream;
- login queues keyed by login ID, including pre-registration buffering;
- turn queues keyed by turn ID, including notifications emitted before `turn/start` returns;
- goal operations keyed by thread ID and spanning runtime-created continuation turns;
- cleanup on terminal notifications and transport failure;
- typed unknown notifications retaining method and raw params.

Implement JSON-RPC error mapping for standard codes, server-range errors, overload metadata,
retry-limit detection, and `errors.Is`/`errors.As` support. Retry must use bounded exponential
backoff with jitter, respect context cancellation, and retry only classified transient failures.

Implement typed server-request dispatch. Command/file approval requests, tool user input, and every
generated server request reach a caller handler. Default decisions match the reference behavior;
unknown requests are surfaced rather than accepted silently.

Exit gate:

- interleaved turn/login/goal notifications route only to their owners;
- early notifications replay in order;
- one slow route cannot corrupt another route;
- typed and unknown server requests receive exactly one response;
- all Python error/retry behavior cases have Go equivalents.

### Phase 4 — public Client, Thread, Turn, and input API

Add an idiomatic root API:

- `NewClient(ctx, ...Option) (*Client, error)` performs launch + initialize;
- `Client.Metadata`, `Close`, and `Wait` expose lifecycle explicitly;
- thread start/list/read/resume/fork/archive/unarchive/name/compact;
- `Thread.Run`, `Thread.StartTurn`, `Thread.Read`, `Thread.SetName`, `Thread.Compact`;
- `Turn.Stream`, `Turn.Run`, `Turn.Steer`, `Turn.Interrupt`;
- model listing and all Python start/turn parameters;
- text, data-URL image, local image, skill, and mention inputs;
- structured output schema without temporary files when app-server accepts JSON directly;
- result fields for ID, status, error, timestamps, duration, final response, items, and usage;
- final response selection honors message phase, not merely the last agent message.

All blocking methods accept `context.Context` as the first parameter. Streaming uses one pull
iterator contract: `stream, err := turn.Stream(ctx)`, `event, err := stream.Next(ctx)`, and
`err = stream.Close()`. `Next` yields exactly one routed notification, returns `io.EOF` on normal
turn completion, and returns the same stable terminal error on subsequent calls after failure.
`Close` is idempotent, unregisters the route, and makes future `Next` calls return an error matching
`ErrStreamClosed`. `Turn.Run` consumes this iterator. Deprecated `RunStreamed` adapts it through a
bounded channel and cancels/closes the iterator when the consumer stops.

Exit gate:

- every curated Python public method/parameter has a documented Go equivalent in the coverage
  matrix;
- compile-time examples cover common, streaming, steer/interrupt, image, and structured output;
- concurrent turns on one client pass race and fake-server tests.

### Phase 5 — account and login parity

Implement:

- reuse of existing Codex authentication;
- API-key login;
- ChatGPT browser login handle with login ID and auth URL;
- device-code login handle with verification URL and user code;
- wait and cancel for both interactive login modes;
- account read with refresh-token option and logout;
- login notification registration before events can be lost;
- context cancellation that unregisters routes and attempts remote cancellation.

Secrets must never appear in errors, logs, debug transcripts, or test snapshots. Environment maps
and API keys are copied defensively and redacted.

Exit gate:

- success, failure, cancellation, concurrent login, early completion, transport close, and redaction
  tests pass;
- real-runtime smoke tests validate account read without mutating credentials; mutating login tests
  run only in explicitly provisioned secret-isolated jobs.

### Phase 6 — goal parity

Port the logical goal state machine:

- require persisted, idle threads before goal start;
- reserve routing before clear/set so generated turns cannot race registration;
- clear old goal, set objective active, wait with a bounded timeout for runtime-generated turn;
- coalesce continuation turns into one logical stream;
- reject a normal turn while a goal operation owns the thread;
- pause and interrupt on cancellation;
- tolerate active-turn replacement during interrupt by retrying with the server-reported ID;
- release state exactly once on completion, failure, timeout, cancellation, or transport close.

Exit gate:

- the Python goal-operation scenarios are ported as black-box fake-server tests;
- concurrent goal starts on one thread serialize; different threads may proceed concurrently;
- no state, goroutine, or route remains after terminal paths.

### Phase 7 — cross-platform runtime packaging and resolution

Supported target table:

| GOOS | GOARCH | upstream triple | executable |
|---|---|---|---|
| darwin | amd64 | x86_64-apple-darwin | codex |
| darwin | arm64 | aarch64-apple-darwin | codex |
| linux | amd64 | x86_64-unknown-linux-musl | codex |
| linux | arm64 | aarch64-unknown-linux-musl | codex |
| windows | amd64 | x86_64-pc-windows-msvc | codex.exe |
| windows | arm64 | aarch64-pc-windows-msvc | codex.exe |

Release artifacts:

- `codex-sdk-go-runtime_<runtime-version>_<goos>_<goarch>.tar.gz` for each target;
- release copy `codex-sdk-go-runtime-manifest.json` of the canonical checked-in
  `runtime/manifest.json`, containing SDK/runtime versions, upstream repository/tag, and for every
  target: GOOS/GOARCH/triple, upstream asset, archive name/hash/size, and each packaged file's
  path/hash/size/mode/role;
- `codex-sdk-go-runtime-manifest.json.sig`, a JSON signature envelope with `keyId`,
  `algorithm: "ed25519"`, and a base64 signature over the exact manifest bytes;
- provenance linking upstream and repackaged hashes.
- SPDX SBOM plus upstream license/NOTICE material for the runtime and every packaged companion;
  repackaging must preserve native code signatures and record signature verification where the
  upstream platform supplies them.

The deterministic manifest schema is versioned with `schemaVersion` and contains `sdkVersion`,
`runtimeVersion`, `upstreamRepo`, `upstreamTag`, and a target array sorted by triple. Trust roots are
versioned public keys embedded in `internal/runtimebin/trust_roots.go`; verification uses the Go
standard library's `crypto/ed25519`. Key rotation ships an overlap release that trusts old and new
keys before a later release removes the old key. Trust-root changes require security review. Online
and offline installation both require the manifest/signature pair and verify it before selecting or
opening an archive. SLSA provenance is additional release evidence, not the installer's trust root.

`internal/runtimebin` resolution order:

1. explicit option;
2. `CODEX_RUNTIME_PATH`;
3. verified SDK-managed cache for the pinned version and target;
4. `codex` on PATH only when explicitly allowed by policy.

`cmd/codex-sdk-runtime install` performs explicit download. It must use HTTPS, verify the manifest
and archive SHA-256 before extraction, reject traversal/absolute paths/symlinks/unexpected files,
apply executable permissions, write through a temporary directory, and atomically rename under a
cross-process lock. `verify`, `path`, and `remove` commands operate only on exact managed versions.

Archives preserve required companion binaries/directories such as `codex-code-mode-host` and make
their directory available to the child process PATH. Offline installation accepts a local archive
plus manifest.

Exit gate:

- build and archive-layout tests pass for all six targets;
- native `codex --version` and app-server initialize/close smoke tests run on all six targets;
- Linux artifacts run on both a glibc distribution and Alpine/musl where supported;
- corrupt, truncated, wrong-target, wrong-version, traversal, symlink, concurrent-install, and
  interrupted-install tests fail safely without damaging an existing cache.
- license/NOTICE, SBOM, provenance, checksum, manifest signature, and available native-signature
  verification pass for every artifact.
- wrong-key, altered-byte, malformed-envelope, and key-rotation signature tests pass; the release
  manifest is byte-for-byte equal to the checked-in canonical manifest.

### Phase 8 — v0.1 compatibility migration

Keep existing exported symbols and behavior compile-compatible during the transition:

- retain `NewCodex`, `CodexOptions`, `StartThread`, `ResumeThread`, `Thread.Run`, and
  `Thread.RunStreamed` as deprecated facade APIs;
- translate existing snake_case JSONL item types to new typed protocol items where possible and
  preserve `UnknownItem.Raw` behavior;
- retain explicit CLI-path/env/config precedence unless documented as a necessary incompatibility;
- allow PATH lookup by default only through the deprecated facade to preserve v0.1 behavior; the
  new client requires an explicit PATH policy;
- remove the data race on thread ID and guarantee safe concurrent reads;
- keep the old exec transport in an internal legacy path only until facade conformance is proven;
- publish a migration guide mapping every old symbol to the new API.

Freeze the old contract before editing transport with these checked-in canaries:

- `testdata/compat/v0.1/public-api.txt`, generated from normalized `go doc -all` output;
- `testdata/compat/v0.1/compile/`, containing current README/examples and representative consumer
  builds;
- `testdata/compat/v0.1/transcripts/`, containing process arguments, environment, JSONL event,
  structured-output, and failure goldens;
- `docs/codex-sdk-parity/compatibility-ledger.md`, recording every preserved behavior and approved
  intentional difference.

The preserved behavior contract includes: CLI resolution `explicit option > PATH >
vendor/<target-triple>/codex`; a non-nil `Env` fully replaces inherited environment while nil
inherits; BaseURL/API-key overrides and originator configuration; option/config/TOML precedence;
structured-output object validation, temporary-file lifetime, and cleanup; start/resume/thread-ID
semantics; `Run`/`RunStreamed` events, final response, usage, and errors; text/local-image inputs;
and raw unknown items. Thread-ID access remains source-compatible but becomes race-free as an
explicit defect fix.

Exit gate:

- current README examples and downstream compile fixtures build unchanged;
- old behavior regression tests pass against the facade;
- intentional differences are listed in migration documentation and release notes;
- removal is deferred to a separately approved compatibility decision.

### Phase 9 — documentation, CI, and release readiness

Documentation deliverables:

- package overview and lifecycle ownership;
- installation and runtime acquisition, including offline and custom-binary workflows;
- API reference and executable examples;
- authentication, approvals, goals, concurrency, cancellation, error/retry, and security guidance;
- generated protocol/versioning policy;
- migration guide and supported-platform table;
- contributor generation/test/release instructions;
- changelog and provenance verification instructions.

CI lanes:

- formatting, `go vet`, unit tests, race tests, coverage, fuzz smoke, examples;
- generator reproducibility and schema drift;
- fake app-server behavioral suite;
- cross-compilation for all targets;
- native packaged-runtime integration matrix for all targets;
- archive security tests and checksum/provenance verification;
- compatibility build against pinned downstream fixtures.

Native runtime jobs publish `artifacts/evidence/<target-triple>.json` containing runner identity,
reported `runtime.GOOS`/`runtime.GOARCH`, archive/manifest hashes, `codex --version`, and
initialize/close results. `runtime-native.yml` validates those fields against the selected manifest
target; cross-compilation or emulation alone cannot satisfy native acceptance.

Exit gate:

- every acceptance criterion in `acceptance-test-spec.md` has evidence attached to a release
  candidate;
- no required matrix job is skipped or marked allowed-to-fail;
- generated and handwritten diffs are reviewed separately;
- release candidate installs and runs from a clean machine on all supported targets.

## 6. Release sequence

1. **Protocol preview:** generated package, manifest, and drift CI; no public transport switch.
2. **Transport preview:** internal app-server transport/router with fake-server tests.
3. **API beta:** new Client/Thread/Turn/login/goal API and compatibility facade.
4. **Runtime RC:** six signed runtime artifacts, installer, native matrix, migration docs.
5. **Parity release:** all acceptance gates green; pin and schema hash recorded in release notes.

Each release must pin one exact runtime version. SDK and runtime pins move together through an
automated update PR containing only schema/generated/runtime-manifest changes plus required
handwritten compatibility adjustments.

## 7. Risks and mitigations

| Risk | Mitigation | Release blocker |
|---|---|---|
| upstream schema changes faster than handwritten API | complete generation manifest and drift CI | yes |
| generated Go identifiers collide | deterministic naming table; fail on collision | yes |
| app-server emits event before route registration | pre-registration buffering with bounded fail-fast overflow | yes |
| blocked reader causes global deadlock | reader never invokes user code; isolated bounded queues | yes |
| abandoned streams leak goroutines | context-owned stream lifecycle and terminal cleanup tests | yes |
| credentials leak through stderr/errors | central redaction and secret canary tests | yes |
| runtime archive is replaced or compromised | pinned hashes, signed manifest, provenance, version check | yes |
| one platform is compile-only | native initialize/close test required for every supported target | yes |
| huge generated diffs hide behavior changes | separate generated/handwritten commits and review lanes | yes |
| compatibility facade changes semantics | black-box v0.1 fixtures and migration ledger | yes |
| runtime cache corruption/concurrent install | verified atomic install with per-version lock | yes |
| unsupported future notification crashes client | raw unknown payload fallback + fuzz tests | yes |

## 8. Staffing and change ownership

Recommended execution lanes after this plan is approved:

- **Architect/executor:** transport lifecycle and root API ownership.
- **Executor:** schema generator and generated protocol package ownership.
- **Executor/security reviewer:** runtime packaging, installer, hashes, extraction, provenance.
- **Executor/test engineer:** fake app-server harness and behavioral parity suite.
- **Executor:** account/login and approval flows.
- **Executor:** goal state machine and concurrency tests.
- **Writer/verifier:** docs, migration, traceability, release evidence.

Shared files (`go.mod`, root public API, generated manifest, CI workflows, coverage matrix) have one
named owner at a time. Workers must not overwrite existing user changes or regenerate protocol
files while another lane is editing generator inputs.

## 9. Completion rule

The rebuild is complete only when every MUST criterion in the acceptance test specification passes
for the pinned runtime on all six supported targets, the coverage matrix has no missing or partial
required row, all generated definitions/methods are accounted for, compatibility fixtures pass,
and release provenance is published. Passing unit tests on one platform is not sufficient.
