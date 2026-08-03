# Generated Protocol Policy

The protocol layer is generated from the pinned upstream schema snapshot and checked into the
repository.

## Generated Artifacts

Key generated files include:

- `protocol/generated_types.go`
- `protocol/generated_registries.go`
- `protocol/manifest.json`
- `schema/*.json`

The source-of-truth pin lives in `reference.lock.json`.

## What To Change

If the upstream schema changes:

1. update the pinned reference checkout
2. regenerate the schema and manifest
3. verify the generator output with `GOWORK=off go run ./cmd/codex-sdk-gen verify`
4. update the compatibility ledger and coverage matrix with evidence links

Do not hand-edit generated protocol files. If a change must alter generated output, change the input
schema or generator inputs and rerun the generator.

## Why This Exists

The SDK depends on a large protocol surface. Keeping the generated layer checked in makes the
source reviewable, makes diffs deterministic, and lets the release pipeline compare current output
with the pinned reference checkout.

## Verification Commands

Use these commands when you touch protocol generation:

```bash
GOWORK=off go run ./cmd/codex-sdk-gen verify
GOWORK=off go test ./cmd/codex-sdk-gen
```

If you change runtime packaging inputs at the same time, also run the runtime verification commands
from [release-verification.md](release-verification.md).
