# Codex Python SDK parity program

This directory is the execution and acceptance contract for rebuilding the Go SDK against the
Codex app-server protocol used by the Python SDK snapshot at
`../../other/codex/sdk/python`.

The target is behavioral and protocol parity, expressed idiomatically in Go. Python's separate
synchronous and asynchronous wrappers map to one context-aware, concurrency-safe Go API. The
current user guide lives in the root [README](../../README.md); the contribution workflow lives in
[CONTRIBUTING.md](../../CONTRIBUTING.md).

## Documents

- [Implementation plan](implementation-plan.md): architecture, work breakdown, migration,
  runtime distribution, release sequence, risks, and ownership boundaries.
- [Acceptance test specification](acceptance-test-spec.md): executable quality gates and the
  definition of done.
- [Coverage matrix](coverage-matrix.md): traceability from Python capabilities and generated
  protocol surfaces to planned Go packages and acceptance tests.
- [v0.1 compatibility ledger](compatibility-ledger.md): frozen v0.2 release history, the approved
  v0.3 facade removal, migration mapping, and acceptance evidence.

The compatibility ledger includes the mechanical migration map from the removed v0.1 facade to
the context-first API.

OMX execution handoff artifacts are stored in:

- `/.omx/plans/prd-codex-sdk-go-parity.md`
- `/.omx/plans/test-spec-codex-sdk-go-parity.md`

## Reference baseline

- Python repository revision: `bb5054fe47` (2026-08-03).
- Python runtime pin: `openai-codex-cli-bin==0.144.4`.
- Go SDK baseline: `a46504d` (`v0.1.2`).
- Supported runtime targets: Darwin, Linux, and Windows on both amd64 and arm64.

Changing the Python revision or runtime pin requires regenerating the protocol manifest, updating
the coverage matrix, and re-running every release gate in the acceptance specification.

The default developer checkout is `../../other/codex`; CI checks out the same locked revision at
`_reference/codex`. `CODEX_REFERENCE_REPO` may select another local checkout, but its `HEAD` must
equal the locked commit. Phase 0 records these values and the aggregate schema hash in
`reference.lock.json`; generation and parity tests fail rather than silently using another source.
