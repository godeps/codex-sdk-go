# Parity coverage and traceability matrix

This matrix is the human-readable scope ledger. `protocol/manifest.json` will be the exhaustive
machine-readable ledger for generated definitions and methods.

Status values during implementation: `missing`, `partial`, `implemented`, `verified`, or `n/a` with
a written language rationale. The parity release requires every required row to be `verified`.

## Functional surface

| ID | Reference behavior | Current Go | Planned owner | Acceptance |
|---|---|---|---|---|
| F-001 | persistent app-server + initialize | implemented | `internal/appserver`, `client.go` | TRN-001 |
| F-002 | explicit close/context ownership | implemented | `client.go`, `internal/appserver` | TRN-005, API-001 |
| F-003 | arbitrary typed JSON-RPC request | implemented | `internal/appserver` | TRN-002, ERR-001 |
| F-004 | concurrent response correlation | implemented | `internal/appserver` | TRN-002/003 |
| F-005 | global notification stream | implemented | `internal/router` | RTR-004 |
| F-006 | early/interleaved turn routing | implemented | `internal/router` | RTR-001 |
| F-007 | login notification routing | implemented | `internal/router`, `login.go` | RTR-002 |
| F-008 | goal notification routing | implemented | `internal/router`, `goal.go` | RTR-003 |
| F-009 | typed server-request/approval handling | implemented | `client.go`, `internal/router` | APR-001/002 |
| F-010 | start thread | implemented | `client.go`, `thread.go` | API-002 |
| F-011 | list/filter/sort/page threads | implemented | `client.go` | API-002 |
| F-012 | read thread/history | implemented | `thread.go` | API-002 |
| F-013 | resume thread with overrides | implemented | `client.go` | API-002 |
| F-014 | fork thread | implemented | `client.go` | API-002 |
| F-015 | archive/unarchive thread | implemented | `client.go` | API-002 |
| F-016 | set thread name | implemented | `thread.go` | API-002 |
| F-017 | compact thread | implemented | `thread.go` | API-002 |
| F-018 | start/collect turn | implemented | `turn.go` | API-003/004 |
| F-019 | routed full event stream | implemented | `turn.go`, `internal/router` | API-003/007 |
| F-020 | steer active turn | implemented | `turn.go` | API-003 |
| F-021 | interrupt active turn | implemented | `turn.go` | API-003 |
| F-022 | rich TurnResult | implemented | `turn.go` | API-003/004 |
| F-023 | text input | implemented | `inputs.go` | API-005 |
| F-024 | data-URL image input | implemented | `inputs.go` | API-005 |
| F-025 | local image input | implemented | `inputs.go` | API-005 |
| F-026 | skill input | implemented | `inputs.go` | API-005 |
| F-027 | mention input | implemented | `inputs.go` | API-005 |
| F-028 | complete turn options/persistence | implemented | `options.go`, `turn.go` | API-006 |
| F-029 | model list + metadata | implemented | `client.go` | API-001 |
| F-030 | typed errors + overload retry | implemented | `errors.go`, `retry.go` | ERR-001/002 |
| F-031 | account read/logout | implemented | `login.go` | AUTH-001/005 |
| F-032 | API-key login | implemented | `login.go` | AUTH-002 |
| F-033 | ChatGPT browser login handle | implemented | `login.go` | AUTH-003 |
| F-034 | device-code login handle | implemented | `login.go` | AUTH-004 |
| F-035 | logical goal start/stream | implemented | `goal.go` | GOAL-001/002/003 |
| F-036 | goal exclusion/cancel/cleanup | implemented | `goal.go` | GOAL-004/005/006 |
| F-037 | future unknown payload tolerance | partial item fallback | `protocol`, `internal/router` | PROTO-005 |
| F-038 | v0.1 public source compatibility | implemented | `compatibility.go` | COMPAT-001/002 |
| F-039 | structured output schema and cleanup | implemented | `turn.go`, compatibility facade | API-006, COMPAT-003 |

## Generated protocol surface

The generator manifest must contain one row for every source entity with:

```json
{
  "sourceKind": "definition|clientRequest|response|notification|serverRequest|unionVariant",
  "wireName": "...",
  "schemaPointer": "#/definitions/...",
  "goSymbol": "...",
  "goFile": "protocol/generated_....go",
  "discriminator": "...",
  "status": "generated|alias|unsupported",
  "tests": ["..."]
}
```

Required generated categories:

| ID | Category | Completeness proof | Acceptance |
|---|---|---|---|
| P-001 | all aggregate v2 definitions | set equality: schema definitions ↔ manifest | PROTO-002 |
| P-002 | all ClientRequest methods and params | set equality + request registry | PROTO-002 |
| P-003 | all response types | request registry response mapping | PROTO-002/004 |
| P-004 | all ServerNotification methods | notification registry set equality | PROTO-002/005 |
| P-005 | all ServerRequest methods | server-request registry set equality | PROTO-002, APR-001 |
| P-006 | all ThreadItem variants | discriminator set equality + unknown fallback | PROTO-002/005 |
| P-007 | all enum values and future values | constants + arbitrary string round-trip | PROTO-004/005 |
| P-008 | all notification route identifiers | generated turn/login/thread routing metadata | RTR-001/002/003 |
| P-009 | schema aliases/required/nullable rules | generated golden vectors | PROTO-004 |
| P-010 | unknown/malformed payload fallback | raw JSON golden + fuzz tests | PROTO-005/006 |

## Runtime and platform surface

| ID | Capability | Planned owner | Acceptance |
|---|---|---|---|
| R-001 | Darwin amd64 asset | implemented | `runtime/manifest.json`, release pipeline | RUN-001/002/008 |
| R-002 | Darwin arm64 asset | implemented | same | RUN-001/002/008 |
| R-003 | Linux amd64 musl asset | implemented | same | RUN-001/002/008 |
| R-004 | Linux arm64 musl asset | implemented | same | RUN-001/002/008 |
| R-005 | Windows amd64 MSVC asset | implemented | same | RUN-001/002/008 |
| R-006 | Windows arm64 MSVC asset | implemented | same | RUN-001/002/008 |
| R-007 | checksum/signature/provenance | implemented | runtime release pipeline | RUN-001/004, REL-002 |
| R-008 | online explicit install | implemented | `cmd/codex-sdk-runtime` | RUN-003/004/005 |
| R-009 | offline local install | implemented | `cmd/codex-sdk-runtime` | RUN-003/004 |
| R-010 | resolution precedence/PATH casing | implemented | `internal/runtimebin` | RUN-006 |
| R-011 | companion executable layout | implemented | packager, `internal/runtimebin` | RUN-007 |
| R-012 | stable/prerelease version mapping | implemented | packager | RUN-009 |
| R-013 | licenses/NOTICE, SPDX SBOM, provenance, native signatures | implemented | runtime release pipeline | RUN-010, REL-002 |
| R-014 | Ed25519 trust roots, signature envelope, rotation | implemented | `internal/runtimebin/trust_roots.go` | RUN-004, REL-002 |
| R-015 | native runner identity/evidence | `runtime-native.yml`, release evidence | RUN-002, RUN-008 |

## Python behavioral test disposition

The pinned Python `sdk/python/tests` directory must be represented by explicit entries in
`testdata/python-test-disposition.json`:

```json
{
  "entries": [
    {
      "python_test": "test_app_server_streaming.py::test_sync_stream_routes_text_deltas_and_completion",
      "category": "behavior|generation|packaging|documentation|python_language_only",
      "status": "equivalent|release_pipeline_equivalent|language_not_applicable",
      "evidence": [
        "go_test:client_api_test.go:TestClientTurnLifecycleAndPayloadMapping",
        "workflow:.github/workflows/runtime-native.yml:aggregate"
      ],
      "rationale": ""
    }
  ]
}
```

Rules:

- every pinned Python test has exactly one explicit row; pattern inheritance is not allowed;
- functional rows must be `equivalent` or `release_pipeline_equivalent`;
- Python language/package-manager mechanics may use `language_not_applicable`, but only with
  category `python_language_only` and a concrete rationale;
- `go_test:` evidence must point to a real `_test.go` file and existing `Test...` symbol;
- `workflow:` evidence must point to a real workflow file and, when present, a real job id;
- QA-004 fails on missing rows, stale rows, fake evidence, or release-pipeline rows without
  workflow evidence.

## Evidence state

Before implementation, all rows remain baseline `missing` or `partial`. Implementers update status
only after attaching the exact test name and CI evidence. Code presence alone is `implemented`, not
`verified`.
