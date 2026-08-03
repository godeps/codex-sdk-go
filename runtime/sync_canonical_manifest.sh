#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: runtime/sync_canonical_manifest.sh <records-dir>" >&2
  exit 1
fi

records_dir=$1
manifest_path="runtime/manifest.json"
signature_path="runtime/manifest.json.sig"

mapfile -t record_paths < <(find "$records_dir" -type f -name 'record.json' | sort)
if [[ ${#record_paths[@]} -ne 6 ]]; then
  echo "expected 6 record.json files under $records_dir, found ${#record_paths[@]}" >&2
  exit 1
fi

manifest_args=(
  manifest
  --sdk-version "v0.2.0"
  --runtime-version "0.144.4"
  --output "$manifest_path"
)
for record_path in "${record_paths[@]}"; do
  manifest_args+=(--record "$record_path")
done

GOWORK=off go run ./cmd/codex-sdk-runtime "${manifest_args[@]}"

if [[ -n "${CODEX_RUNTIME_MANIFEST_PRIVATE_KEY_BASE64:-}" ]]; then
  GOWORK=off go run ./cmd/codex-sdk-runtime sign \
    --manifest "$manifest_path" \
    --output "$signature_path" \
    --key-id "runtime-manifest-v3"
else
  echo "CODEX_RUNTIME_MANIFEST_PRIVATE_KEY_BASE64 is not set; wrote $manifest_path without $signature_path" >&2
fi
