# Parity coverage and traceability matrix

This matrix is the human-readable scope ledger. `protocol/manifest.json` will be the exhaustive
machine-readable ledger for generated definitions and methods.

Status values during implementation: `missing`, `partial`, `implemented`, `verified`, or `n/a` with
a written language rationale. The parity release requires every required row to be `verified`.

## Functional surface

| ID | Reference behavior | Status | Owner | Acceptance evidence |
|---|---|---|---|---|
| F-001 | persistent app-server + initialize | verified | `internal/appserver`, `client.go` | TRN-001 |
| F-002 | explicit close/context ownership | verified | `client.go`, `internal/appserver` | TRN-005, API-001 |
| F-003 | arbitrary typed JSON-RPC request | verified | `internal/appserver` | TRN-002, ERR-001 |
| F-004 | concurrent response correlation | verified | `internal/appserver` | TRN-002/003 |
| F-005 | global notification stream | verified | `internal/router` | RTR-004 |
| F-006 | early/interleaved turn routing | verified | `internal/router` | RTR-001 |
| F-007 | login notification routing | verified | `internal/router`, `login.go` | RTR-002 |
| F-008 | goal notification routing | verified | `internal/router`, `goal.go` | RTR-003 |
| F-009 | typed server-request/approval handling | verified | `client.go`, `internal/router` | APR-001/002 |
| F-010 | start thread | verified | `client.go`, `thread.go` | API-002 |
| F-011 | list/filter/sort/page threads | verified | `client.go` | API-002 |
| F-012 | read thread/history | verified | `thread.go` | API-002 |
| F-013 | resume thread with overrides | verified | `client.go` | API-002 |
| F-014 | fork thread | verified | `client.go` | API-002 |
| F-015 | archive/unarchive thread | verified | `client.go` | API-002 |
| F-016 | set thread name | verified | `thread.go` | API-002 |
| F-017 | compact thread | verified | `thread.go` | API-002 |
| F-018 | start/collect turn | verified | `turn.go` | API-003/004 |
| F-019 | routed full event stream | verified | `turn.go`, `internal/router` | API-003/007 |
| F-020 | steer active turn | verified | `turn.go` | API-003 |
| F-021 | interrupt active turn | verified | `turn.go` | API-003 |
| F-022 | rich TurnResult | verified | `turn.go` | API-003/004 |
| F-023 | text input | verified | `inputs.go` | API-005 |
| F-024 | data-URL image input | verified | `inputs.go` | API-005 |
| F-025 | local image input | verified | `inputs.go` | API-005 |
| F-026 | skill input | verified | `inputs.go` | API-005 |
| F-027 | mention input | verified | `inputs.go` | API-005 |
| F-028 | complete turn options/persistence | verified | `options.go`, `turn.go` | API-006 |
| F-029 | model list + metadata | verified | `client.go` | API-001 |
| F-030 | typed errors + overload retry | verified | `errors.go`, `retry.go` | ERR-001/002 |
| F-031 | account read/logout | verified | `login.go` | AUTH-001/005 |
| F-032 | API-key login | verified | `login.go` | AUTH-002 |
| F-033 | ChatGPT browser login handle | verified | `login.go` | AUTH-003 |
| F-034 | device-code login handle | verified | `login.go` | AUTH-004 |
| F-035 | logical goal start/stream | verified | `goal.go` | GOAL-001/002/003 |
| F-036 | goal exclusion/cancel/cleanup | verified | `goal.go` | GOAL-004/005/006 |
| F-037 | future unknown payload tolerance | verified | `protocol`, `internal/router` | PROTO-005: raw round-trip tests for item, notification, primitive, user-input, and nested-union fallbacks |
| F-038 | v0.1 facade removal and migration | verified | `public_api_surface_test.go`, migration ledger | COMPAT-001/002/004 |
| F-039 | structured output schema validation | verified | `turn_api.go`, `output_schema.go` | API-006, COMPAT-003 |

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

| ID | Category | Status | Completeness proof | Acceptance |
|---|---|---|---|---|
| P-001 | all aggregate v2 definitions | verified | set equality: schema definitions ↔ manifest | PROTO-002 |
| P-002 | all ClientRequest methods and params | verified | set equality + request registry | PROTO-002 |
| P-003 | all response types | verified | request registry response mapping | PROTO-002/004 |
| P-004 | all ServerNotification methods | verified | notification registry set equality | PROTO-002/005 |
| P-005 | all ServerRequest methods | verified | server-request registry set equality | PROTO-002, APR-001 |
| P-006 | all ThreadItem variants | verified | discriminator set equality + unknown fallback | PROTO-002/005 |
| P-007 | all enum values and future values | verified | constants + arbitrary string round-trip | PROTO-004/005 |
| P-008 | all notification route identifiers | verified | generated turn/login/thread routing metadata | RTR-001/002/003 |
| P-009 | schema aliases/required/nullable rules | verified | generated golden vectors | PROTO-004 |
| P-010 | unknown/malformed payload fallback | verified | raw JSON golden + seven fuzz targets | PROTO-005/006 |

## Runtime and platform surface

| ID | Capability | Status | Owner | Acceptance evidence |
|---|---|---|---|---|
| R-001 | Darwin amd64 asset | verified | `runtime/manifest.json`, release pipeline | RUN-001/002/008 |
| R-002 | Darwin arm64 asset | verified | same | RUN-001/002/008 |
| R-003 | Linux amd64 musl asset | verified | same | RUN-001/002/008 |
| R-004 | Linux arm64 musl asset | verified | same | RUN-001/002/008 |
| R-005 | Windows amd64 MSVC asset | verified | same | RUN-001/002/008 |
| R-006 | Windows arm64 MSVC asset | verified | same | RUN-001/002/008 |
| R-007 | checksum/signature/provenance | verified | runtime release pipeline | RUN-001/004, REL-002 |
| R-008 | online explicit install | verified | `cmd/codex-sdk-runtime` | RUN-003/004/005 |
| R-009 | offline local install | verified | `cmd/codex-sdk-runtime` | RUN-003/004 |
| R-010 | resolution precedence/PATH casing | verified | `internal/runtimebin` | RUN-006 |
| R-011 | companion executable layout | verified | packager, `internal/runtimebin` | RUN-007 |
| R-012 | stable/prerelease version mapping | verified | packager | RUN-009 |
| R-013 | licenses/NOTICE, SPDX SBOM, provenance, native signatures | verified | runtime release pipeline | RUN-010, REL-002 |
| R-014 | Ed25519 trust roots, signature envelope, rotation | verified | `internal/runtimebin/trust_roots.go` | RUN-004, REL-002 |
| R-015 | native runner identity/evidence | verified | `runtime-native.yml`, release evidence | RUN-002, RUN-008; six native jobs in run 30813943748 |

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

Rows move from `implemented` to `verified` only when their named acceptance test and exact-SHA CI or
release evidence pass. The final release commit must contain no required `missing` or `partial` row.
