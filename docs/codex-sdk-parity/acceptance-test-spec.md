# Acceptance test specification

This document is the release-blocking definition of done for full Python SDK behavioral and
protocol parity. Every requirement marked **MUST** needs reproducible evidence from the same release
candidate and pinned runtime. A skipped or allowed-to-fail required job is a failure.

## 1. Test environments

### 1.1 Required target matrix

| Target ID | GOOS/GOARCH | runtime asset | required execution |
|---|---|---|---|
| T-DARWIN-AMD64 | darwin/amd64 | x86_64-apple-darwin | native |
| T-DARWIN-ARM64 | darwin/arm64 | aarch64-apple-darwin | native |
| T-LINUX-AMD64 | linux/amd64 | x86_64-unknown-linux-musl | native glibc + Alpine/musl |
| T-LINUX-ARM64 | linux/arm64 | aarch64-unknown-linux-musl | native; emulation is supplemental |
| T-WINDOWS-AMD64 | windows/amd64 | x86_64-pc-windows-msvc | native |
| T-WINDOWS-ARM64 | windows/arm64 | aarch64-pc-windows-msvc | native; cross-compile is supplemental |

If hosted native runners are unavailable, a controlled self-hosted runner is required. Cross-build
or QEMU-only evidence cannot close a native execution requirement.

### 1.2 Test layers

| Layer | Runtime | Purpose |
|---|---|---|
| Unit | none | data types, generation, mapping, state machines, security primitives |
| Contract | fake app-server | exact JSON-RPC behavior, races, errors, concurrency |
| Integration | pinned real runtime | initialize, public workflows, binary layout |
| Packaging | release artifacts | download, verify, extract, install, resolve, provenance |
| Compatibility | fake + real runtime | unchanged v0.1 source examples and semantics |

## 2. Protocol generation acceptance

### PROTO-001 — exact source pin

**MUST:** `reference.lock.json`, `schema/`, and `protocol/manifest.json` record upstream repository
and commit `bb5054fe47`, Python root `sdk/python`, runtime version `0.144.4`, upstream release tag,
aggregate schema filename/SHA-256, generator version, and generation command. Reference resolution
accepts only a checkout whose `HEAD` and metadata match the lock.

Evidence:

```bash
GOWORK=off go run ./cmd/codex-sdk-gen verify
```

The command fails for a different runtime, schema hash, or missing metadata.

### PROTO-002 — complete definition coverage

**MUST:** every definition in the aggregate v2 bundle maps to a generated Go symbol or an explicit
generated alias; every client request, response, server notification, and server request maps to a
registry entry. The manifest reports:

- `missingDefinitions: []`
- `missingMethods: []`
- `unsupportedConstructs: []`
- `duplicateSymbols: []`
- exact source and generated counts.

The test compares sets, not a hand-maintained expected count.

### PROTO-003 — deterministic output

**MUST:** two clean generations on Linux and one clean generation on Windows produce byte-identical
Go files and manifests after `gofmt`.

```bash
GOWORK=off go generate ./protocol/...
git diff --exit-code -- schema protocol
```

### PROTO-004 — JSON aliases and optionality

**MUST:** generated golden tests prove camelCase wire aliases, required-vs-optional fields,
required-nullable fields, arrays/maps, numeric values, and nested unions match schema examples.

### PROTO-005 — forward compatibility

**MUST:** future enum strings round-trip unchanged. Unknown notification methods, item
discriminators, union variants, and malformed known payloads become typed raw fallback values with
the original JSON retained; they do not terminate the reader.

### PROTO-006 — fuzz safety

**MUST:** fuzz targets for every generated union family and JSON-RPC envelope run for at least
10 minutes in scheduled CI without panic, unbounded allocation, or data loss after successful
round-trip.

## 3. Transport and lifecycle acceptance

### TRN-001 — launch and initialize

**MUST:** `NewClient` launches exactly one app-server process, sends `initialize`, validates nonempty
metadata, sends `initialized`, and returns a ready client. Initialization failure terminates and
reaps the child before returning.

### TRN-002 — request correlation

**MUST:** 1,000 concurrent requests with shuffled responses return to the correct caller. Duplicate,
unknown, and late response IDs do not block or corrupt active requests.

### TRN-003 — single-reader routing

**MUST:** exactly one goroutine reads stdout. User handlers never execute on that reader goroutine.
Interleaved responses, server requests, and notifications remain ordered within their route.

### TRN-004 — cancellation

**MUST:** canceling a request removes its waiter promptly and returns a context-compatible error.
A late server response is ignored safely. Canceling one request does not cancel another or kill the
client.

### TRN-005 — close and failure

**MUST:** `Close` is concurrent-safe and idempotent. It closes input, waits a bounded interval,
terminates, then kills if required. All pending callers and stream consumers receive one terminal
error. `Wait` returns the stable terminal result. No zombie process remains.

### TRN-006 — bounded resources

**MUST:** configured limits exist and default to: 16 MiB line; 400-line/2 MiB stderr; 1,024
in-flight RPCs; 256-event/8 MiB per route; 1,024-event/32 MiB global; 4,096-event/64 MiB total;
64 concurrent server-request handlers; 2-second graceful and 2-second forced-kill waits. Boundary
and first-overflow tests prove route overflow isolates that route, while line/global/total/in-flight
overflow terminates transport. Errors match `ErrLimitExceeded` and, after transport termination,
`ErrTransportClosed`; no event is silently lost.

### TRN-007 — malformed transport

**MUST:** empty stdout, invalid JSON, non-object JSON, oversized lines, truncated lines, write
failure, and nonzero child exit include a redacted bounded stderr tail and wake all waiters.

### TRN-008 — leak and race freedom

**MUST:** unit and contract suites pass `go test -race -count=20`; each lifecycle test proves all SDK
goroutines and fake child processes have stopped using owned wait groups/process handles rather than
timing guesses.

## 4. Router, approval, error, and retry acceptance

### RTR-001 — turn isolation

**MUST:** at least 20 concurrently active turns receive only their own notifications, including when
notifications arrive before `turn/start` returns.

### RTR-002 — login isolation

**MUST:** concurrent interactive logins route completion by login ID. Completion before handle
construction is replayed. Cancel followed by `Wait` returns a stable documented terminal outcome.

### RTR-003 — goal isolation

**MUST:** goal notifications route by thread ID and current physical turn ID. Different thread goals
may progress concurrently; two goals on one thread cannot both acquire ownership.

### RTR-004 — transport failure broadcast

**MUST:** a transport failure wakes every response waiter, login handle, turn stream, goal operation,
and global notification consumer exactly once.

### APR-001 — all server requests handled

**MUST:** every generated server-request method reaches the typed handler registry. Command and file
approval requests, tool user input, and unknown server requests receive exactly one response.
Unknown methods are never auto-approved.

### APR-002 — approval modes

**MUST:** Go's deny-all and auto-review modes serialize to the same app-server approval policy and
reviewer values as Python. Resume/fork/turn omission preserves stored settings; explicit overrides
persist exactly as the reference tests specify.

### ERR-001 — typed errors

**MUST:** JSON-RPC standard codes and server error range map to typed errors containing code, message,
and raw data. Callers can use `errors.As`; wrapped context remains available.

### ERR-002 — retry

**MUST:** only overload-classified errors retry. Maximum attempts, exponential delay cap, jitter
range, context cancellation during sleep, retry-limit errors, and terminal error identity are
deterministically tested with an injected clock/random source.

## 5. Public workflow acceptance

### API-001 — client lifecycle

**MUST:** Client exposes metadata, model listing, explicit close/wait, and safe concurrent use. Every
blocking operation accepts context.

### API-002 — thread lifecycle

**MUST:** start, list with every filter/cursor/sort field, read with and without turns, resume, fork,
archive, unarchive, set name, and compact pass contract and real-runtime tests.

### API-003 — turn lifecycle

**MUST:** start, stream, collect/run, steer, and interrupt work. Collected results expose ID, status,
error, start/completion timestamps, duration, final response, items, and usage.

The stream is a pull iterator. `Next(ctx)` yields one notification, returns `io.EOF` after normal
completion, and returns a stable terminal error after failure. `Close` is concurrent-safe and
idempotent, unregisters its route, and future `Next` calls match `ErrStreamClosed`. `Run` consumes
the iterator; deprecated `RunStreamed` uses a bounded adapter that closes/cancels when abandoned.

### API-004 — final response semantics

**MUST:** collection chooses the latest `final_answer` phase agent message; a phase-less agent
message is fallback only. Failed turns return their server message; missing completion is an error.

### API-005 — inputs

**MUST:** text, data-URL image, local image, skill, and mention inputs serialize exactly. Mixed ordered
input is preserved. Invalid and zero-value input returns a typed error; no item is silently dropped.

### API-006 — turn options

**MUST:** approval mode, cwd, reasoning effort, model, output schema, personality, sandbox policy,
service tier, and reasoning summary map exactly and have omission/persistence tests.

### API-007 — concurrency

**MUST:** one client can list models while multiple turns stream. Concurrent operations on one thread
follow documented server constraints; thread ID/state reads have no data race.

## 6. Login/account acceptance

### AUTH-001 — existing account

**MUST:** the SDK can read the existing account without changing credentials and can request token
refresh according to the protocol.

### AUTH-002 — API key login

**MUST:** API key login uses the exact RPC payload and never exposes the key in errors, stderr,
formatted structs, traces, or fixtures. Canary-secret tests scan all captured output.

### AUTH-003 — browser login

**MUST:** browser login returns login ID + auth URL and supports wait, success/failure notification,
cancel, context cancellation, early completion, and transport close.

### AUTH-004 — device-code login

**MUST:** device login returns login ID + verification URL + user code and satisfies the same wait,
cancel, and failure cases as AUTH-003.

### AUTH-005 — logout

**MUST:** logout uses the expected RPC and subsequent account read reflects the runtime result.
Mutating auth integration tests run only in isolated accounts and never on developer credentials.

## 7. Goal acceptance

### GOAL-001 — preconditions

**MUST:** goal start rejects non-idle, ephemeral, or non-persisted threads before changing state.

### GOAL-002 — race-free start

**MUST:** route reservation happens before clear/set. A physical turn emitted immediately by
`thread/goal/set` is observed, and start times out cleanly when none arrives.

### GOAL-003 — continuation coalescing

**MUST:** multiple runtime continuation turns are exposed as one logical goal stream in order, while
retaining physical turn IDs for diagnostics and interruption.

### GOAL-004 — mutual exclusion

**MUST:** normal turn start and second goal start fail while a goal owns the thread. The lock is
released after every terminal path.

### GOAL-005 — cancel/pause/interrupt

**MUST:** cancellation performs best-effort pause then interrupt. If the active turn changes, retry
uses the server-reported/current turn ID. Errors during cleanup do not leak the route.

### GOAL-006 — terminal cleanup

**MUST:** success, failure, timeout, cancellation, malformed event, and transport close each finish
and unregister state exactly once with no goroutine or waiter leak.

## 8. Runtime packaging acceptance

### RUN-001 — target manifest

**MUST:** canonical `runtime/manifest.json` has exactly one entry for each required target, no
duplicate target, the exact pinned runtime, upstream asset name, archive hash/size, executable,
per-file path/hash/size/mode/role, companion layout, and provenance. The release copy
`codex-sdk-go-runtime-manifest.json` is byte-identical.

### RUN-002 — deterministic packaging

**MUST:** rebuilding a target archive from the same upstream input produces identical file content,
permissions, normalized metadata, manifest record, and checksum.

### RUN-003 — explicit install only

**MUST:** constructing or using a client never downloads implicitly. Missing runtime returns an
actionable error. Explicit install supports online and local/offline archives.

### RUN-004 — secure download and extraction

**MUST:** HTTPS, Ed25519 manifest-envelope verification before asset use, checksum validation,
target/version check, size limit, traversal rejection, absolute-path rejection,
symlink/hardlink/device rejection, allowlisted archive layout, and executable permission tests
pass. The envelope contains recognized `keyId`, `algorithm: "ed25519"`, and base64 signature;
altered exact bytes, wrong key, malformed signature, and untrusted keys fail. Rotation tests prove
one overlap release accepts both old and new keys and a later release rejects the retired key.

### RUN-005 — atomic cache

**MUST:** concurrent installs converge on one verified cache. Interrupted or corrupt installs leave
the previous version usable and no partially selected runtime. Remove only affects an exact managed
version/target.

### RUN-006 — resolution precedence

**MUST:** explicit option > `CODEX_RUNTIME_PATH` > verified managed cache > explicitly allowed PATH.
Broken high-priority selections return errors rather than silently falling back. Windows environment
lookup preserves existing `PATH`/`Path` casing without duplicate injection.

### RUN-007 — archive completeness

**MUST:** every artifact contains executable, metadata, and all companion PATH files required by the
upstream package. `codex-code-mode-host` discovery is tested when present.

### RUN-008 — native smoke

**MUST:** on every target, installed runtime passes version, app-server initialize, model list,
thread start, one streamed turn, and clean close. Network-dependent turn tests use controlled
credentials/endpoints; initialize/close remains offline-capable where runtime permits.
Each native job publishes `artifacts/evidence/<target-triple>.json` with runner identity,
`runtime.GOOS`/`runtime.GOARCH`, manifest/archive hashes, version, and smoke results. Target fields
must equal the manifest entry; cross-build or emulation evidence cannot replace them.

### RUN-009 — version conversion

**MUST:** stable, alpha, beta, and release-candidate upstream tags normalize deterministically to SDK
runtime versions and back; invalid tags fail. Version conversion has golden tests independent of
Python PEP 440 naming.

### RUN-010 — licenses, SBOM, and native signatures

**MUST:** every runtime artifact contains the required upstream license/NOTICE files and has an SPDX
SBOM tied to its checksum. Provenance identifies every upstream input. Repackaging preserves native
binary signatures; macOS and Windows signature verification is recorded when upstream supplies a
signature. Missing legal metadata, SBOM, provenance, or an expected signature blocks release.

## 9. Compatibility acceptance

### COMPAT-001 — source compatibility

**MUST:** compile fixtures using the v0.1 public constructors, options, input helpers, item type
assertions, `Run`, and `RunStreamed` compile unchanged.

### COMPAT-002 — behavior compatibility

**MUST:** black-box goldens cover CLI resolution `explicit > PATH > vendor/<triple>/codex`, nil
environment inheritance versus non-nil full replacement, BaseURL/API-key/originator and
config/TOML precedence, output-schema object validation/temp-file cleanup, start/resume/thread ID,
`Run`/`RunStreamed` events/final response/usage/errors, text/local-image inputs, cancellation,
process error, and `UnknownItem.Raw`. The thread-ID race fix preserves source/API output while
proving concurrent access safe.

### COMPAT-003 — documented differences

**MUST:** every intentional semantic change has old behavior, new behavior, migration, and release
version in a compatibility ledger. Undocumented differences block release.

### COMPAT-004 — deprecation

**MUST:** deprecated APIs remain tested and documented for the promised window. Removal requires a
separate approved plan and, if module version is v1+, semantic-version-compatible release handling.

## 10. Quality, documentation, and release gates

### QA-001 — standard verification

**MUST:** all pass:

```bash
GOWORK=off gofmt -d .
GOWORK=off go vet ./...
GOWORK=off go test ./...
GOWORK=off go test -race ./...
GOWORK=off go test -tags=integration ./...
```

Any configured linter/static analyzer is also required with zero unsuppressed findings.

### QA-002 — coverage

**MUST:** handwritten code has at least 85% statement coverage; `internal/appserver`, `internal/router`,
and `internal/runtimebin` each have at least 90%. Generated lines are excluded from percentage gates
but covered by PROTO-002 through PROTO-006 and golden tests. Coverage cannot be raised by excluding
handwritten error paths.

### QA-003 — repeated and stress execution

**MUST:** concurrency/lifecycle suites pass `-race -count=20`; scheduled stress tests run at least
30 minutes without deadlock, leak, event misrouting, or unbounded memory growth.

### QA-004 — Python scenario parity

**MUST:** every behavioral test in the pinned Python suite is classified as ported, language-not-
applicable with rationale, or release/pipeline equivalent. No functional scenario may be marked
not-applicable solely because it is difficult to reproduce.

### DOC-001 — required docs

**MUST:** README, package docs, API reference, executable examples, runtime installation/offline
guide, auth/approval/goal guide, concurrency/cancellation guide, generated protocol policy,
migration guide, supported-platform table, contributing guide, and release verification guide are
present and tested where executable.

### REL-001 — clean release build

**MUST:** from clean source and empty caches, the source module builds, all six runtime artifacts
package, manifests/signatures/provenance verify, and target-native smoke jobs pass.

### REL-002 — release evidence bundle

**MUST:** publish machine-readable evidence containing commit, Go version, runtime pin, schema hash,
generator hash, test commands/results, coverage, target job URLs, artifact hashes, provenance, known
limitations, and the completed coverage matrix.

### REL-003 — zero-known-error rule

**MUST:** no open blocker/critical defect, test failure, race, generator drift, missing required
platform, unverified artifact, or undocumented compatibility break remains. Lower-severity accepted
risks require owner, rationale, expiry/review date, and issue link.

## 11. Final release checklist

- [ ] PROTO-001 through PROTO-006 pass.
- [ ] TRN-001 through TRN-008 pass.
- [ ] RTR-001 through RTR-004, APR-001/002, and ERR-001/002 pass.
- [ ] API-001 through API-007 pass.
- [ ] AUTH-001 through AUTH-005 pass.
- [ ] GOAL-001 through GOAL-006 pass.
- [ ] RUN-001 through RUN-010 pass on the required target matrix.
- [ ] COMPAT-001 through COMPAT-004 pass.
- [ ] QA-001 through QA-004, DOC-001, and REL-001 through REL-003 pass.
- [ ] Coverage matrix has no required `missing` or `partial` status.
- [ ] A verifier independent of implementation reviewed the evidence bundle.
